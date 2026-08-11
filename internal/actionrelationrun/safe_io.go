package actionrelationrun

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/sys/unix"
)

// openParentNoFollow walks an absolute path one directory descriptor at a
// time. O_NOFOLLOW on every component prevents a pre-existing .nous (or any
// other ancestor) symlink from redirecting panel authority outside its root.
func openParentNoFollow(path string, createParents bool, directoryMode os.FileMode) (int, string, error) {
	clean := filepath.Clean(path)
	if runtime.GOOS == "darwin" {
		switch {
		case clean == "/var" || strings.HasPrefix(clean, "/var/"):
			clean = "/private" + clean
		case clean == "/tmp" || strings.HasPrefix(clean, "/tmp/"):
			clean = "/private" + clean
		}
	}
	if !filepath.IsAbs(clean) || clean == string(filepath.Separator) {
		return -1, "", fmt.Errorf("no-follow path must name an absolute file")
	}
	parts := strings.Split(strings.TrimPrefix(clean, string(filepath.Separator)), string(filepath.Separator))
	if len(parts) < 1 || parts[len(parts)-1] == "" {
		return -1, "", fmt.Errorf("invalid no-follow path")
	}
	fd, err := unix.Open(string(filepath.Separator), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return -1, "", err
	}
	for _, part := range parts[:len(parts)-1] {
		if part == "" || part == "." || part == ".." {
			unix.Close(fd)
			return -1, "", fmt.Errorf("invalid no-follow path component")
		}
		next, openErr := unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if openErr != nil && errors.Is(openErr, unix.ENOENT) && createParents {
			if mkdirErr := unix.Mkdirat(fd, part, uint32(directoryMode.Perm())); mkdirErr != nil && !errors.Is(mkdirErr, unix.EEXIST) {
				unix.Close(fd)
				return -1, "", mkdirErr
			}
			next, openErr = unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		}
		unix.Close(fd)
		if openErr != nil {
			return -1, "", fmt.Errorf("unsafe authority ancestor %q: %w", part, openErr)
		}
		fd = next
	}
	leaf := parts[len(parts)-1]
	if leaf == "" || leaf == "." || leaf == ".." {
		unix.Close(fd)
		return -1, "", fmt.Errorf("invalid no-follow leaf")
	}
	return fd, leaf, nil
}

func ensureDirectoryNoFollow(path string, mode os.FileMode) error {
	parent, _, err := openParentNoFollow(filepath.Join(path, ".sentinel"), true, mode)
	if err != nil {
		return err
	}
	return unix.Close(parent)
}

func checkDirectoryNoFollow(path string) error {
	parent, _, err := openParentNoFollow(filepath.Join(path, ".sentinel"), false, 0)
	if err != nil {
		return err
	}
	return unix.Close(parent)
}

func requireAbsentNoFollow(path string) error {
	parent, leaf, err := openParentNoFollow(path, false, 0)
	if err != nil {
		if errors.Is(err, unix.ENOENT) {
			return nil
		}
		return err
	}
	defer unix.Close(parent)
	var stat unix.Stat_t
	err = unix.Fstatat(parent, leaf, &stat, unix.AT_SYMLINK_NOFOLLOW)
	if errors.Is(err, unix.ENOENT) {
		return nil
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("path already exists: %s", path)
}

func readRegularNoFollow(path string, mode os.FileMode) ([]byte, error) {
	parent, leaf, err := openParentNoFollow(path, false, 0)
	if err != nil {
		return nil, err
	}
	defer unix.Close(parent)
	fd, err := unix.Openat(parent, leaf, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	defer file.Close()
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil || stat.Mode&unix.S_IFMT != unix.S_IFREG || os.FileMode(stat.Mode).Perm() != mode.Perm() || stat.Nlink != 1 {
		return nil, fmt.Errorf("no-follow path is not an exclusive mode-%04o regular file", mode.Perm())
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func readRegularAt(parent int, leaf, path string, mode os.FileMode, allowLinks bool) ([]byte, unix.Stat_t, error) {
	fd, err := unix.Openat(parent, leaf, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, unix.Stat_t{}, err
	}
	file := os.NewFile(uintptr(fd), path)
	defer file.Close()
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil || stat.Mode&unix.S_IFMT != unix.S_IFREG || os.FileMode(stat.Mode).Perm() != mode.Perm() || !allowLinks && stat.Nlink != 1 || allowLinks && stat.Nlink < 1 {
		return nil, unix.Stat_t{}, fmt.Errorf("no-follow path is not a mode-%04o regular file with valid links", mode.Perm())
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, unix.Stat_t{}, err
	}
	return data, stat, nil
}

func readRegularNoFollowAllowLinks(path string, mode os.FileMode) ([]byte, unix.Stat_t, error) {
	parent, leaf, err := openParentNoFollow(path, false, 0)
	if err != nil {
		return nil, unix.Stat_t{}, err
	}
	defer unix.Close(parent)
	return readRegularAt(parent, leaf, path, mode, true)
}

// installAtomicNoFollow makes a fully written and fsynced same-directory
// temporary inode visible with one no-replace link. The content-derived
// temporary basename makes both pre-link retries and post-link cleanup
// deterministic after interruption.
func installAtomicNoFollow(path string, data []byte, mode, directoryMode os.FileMode) (bool, error) {
	parent, leaf, err := openParentNoFollow(path, true, directoryMode)
	if err != nil {
		return false, err
	}
	defer unix.Close(parent)
	digest := sha256.Sum256(data)
	temporary := "." + leaf + ".tmp-" + hex.EncodeToString(digest[:])

	finalData, finalStat, finalErr := readRegularAt(parent, leaf, path, mode, true)
	if finalErr == nil {
		if !bytes.Equal(finalData, data) {
			return false, fmt.Errorf("authority path already exists with different bytes: %s", path)
		}
		tempData, tempStat, tempErr := readRegularAt(parent, temporary, path+" temporary", mode, true)
		switch {
		case errors.Is(tempErr, unix.ENOENT) && finalStat.Nlink == 1:
			return false, nil
		case tempErr == nil && bytes.Equal(tempData, data) && tempStat.Dev == finalStat.Dev && tempStat.Ino == finalStat.Ino && finalStat.Nlink == 2:
			if err := unix.Unlinkat(parent, temporary, 0); err != nil {
				return false, err
			}
			return false, unix.Fsync(parent)
		default:
			return false, fmt.Errorf("authority path has unreconciled hard links: %s", path)
		}
	}
	if !errors.Is(finalErr, unix.ENOENT) {
		return false, finalErr
	}

	tempData, tempStat, tempErr := readRegularAt(parent, temporary, path+" temporary", mode, true)
	if tempErr == nil {
		if !bytes.Equal(tempData, data) || tempStat.Nlink != 1 {
			return false, fmt.Errorf("authority temporary path changed: %s", path)
		}
	} else if errors.Is(tempErr, unix.ENOENT) {
		fd, openErr := unix.Openat(parent, temporary, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, uint32(mode.Perm()))
		if openErr != nil {
			return false, openErr
		}
		file := os.NewFile(uintptr(fd), path+" temporary")
		writeErr := file.Chmod(mode.Perm())
		if writeErr == nil {
			_, writeErr = file.Write(data)
		}
		if writeErr == nil {
			writeErr = file.Sync()
		}
		closeErr := file.Close()
		if writeErr != nil {
			return false, writeErr
		}
		if closeErr != nil {
			return false, closeErr
		}
		tempData, tempStat, tempErr = readRegularAt(parent, temporary, path+" temporary", mode, false)
		if tempErr != nil || !bytes.Equal(tempData, data) || tempStat.Nlink != 1 {
			return false, fmt.Errorf("authority temporary readback mismatch: %s", path)
		}
	} else {
		return false, tempErr
	}

	if err := unix.Linkat(parent, temporary, parent, leaf, 0); err != nil {
		if errors.Is(err, unix.EEXIST) {
			return false, fmt.Errorf("authority path concurrently installed: %s", path)
		}
		return false, err
	}
	linked := true
	syncErr := unix.Fsync(parent)
	unlinkErr := unix.Unlinkat(parent, temporary, 0)
	cleanupSyncErr := unix.Fsync(parent)
	return linked, errors.Join(syncErr, unlinkErr, cleanupSyncErr)
}

func removeExpectedNoFollow(path string, data []byte, mode os.FileMode) error {
	parent, leaf, err := openParentNoFollow(path, false, 0)
	if err != nil {
		if errors.Is(err, unix.ENOENT) {
			return nil
		}
		return err
	}
	defer unix.Close(parent)
	existing, _, err := readRegularAt(parent, leaf, path, mode, true)
	if errors.Is(err, unix.ENOENT) {
		return nil
	}
	if err != nil || !bytes.Equal(existing, data) {
		return fmt.Errorf("refuse to remove changed temporary authority: %s", path)
	}
	if err := unix.Unlinkat(parent, leaf, 0); err != nil {
		return err
	}
	return unix.Fsync(parent)
}

func linkStagedNoFollow(stagedPath, finalPath string, data []byte, mode os.FileMode) (bool, error) {
	if filepath.Dir(stagedPath) != filepath.Dir(finalPath) {
		return false, fmt.Errorf("staged and final authority must share a directory")
	}
	parent, stagedLeaf, err := openParentNoFollow(stagedPath, false, 0)
	if err != nil {
		return false, err
	}
	defer unix.Close(parent)
	finalLeaf := filepath.Base(finalPath)
	staged, stagedStat, err := readRegularAt(parent, stagedLeaf, stagedPath, mode, true)
	if err != nil || !bytes.Equal(staged, data) {
		return false, fmt.Errorf("staged authority does not match expected bytes: %s", stagedPath)
	}
	final, finalStat, finalErr := readRegularAt(parent, finalLeaf, finalPath, mode, true)
	if finalErr == nil {
		if !bytes.Equal(final, data) {
			return false, fmt.Errorf("final authority differs from staged bytes: %s", finalPath)
		}
		if stagedStat.Dev == finalStat.Dev && stagedStat.Ino == finalStat.Ino {
			if err := unix.Unlinkat(parent, stagedLeaf, 0); err != nil {
				return true, err
			}
			return true, unix.Fsync(parent)
		}
		if err := unix.Unlinkat(parent, stagedLeaf, 0); err != nil {
			return true, err
		}
		return true, unix.Fsync(parent)
	}
	if !errors.Is(finalErr, unix.ENOENT) {
		return false, finalErr
	}
	if stagedStat.Nlink != 1 {
		return false, fmt.Errorf("staged authority has unexpected links: %s", stagedPath)
	}
	if err := unix.Linkat(parent, stagedLeaf, parent, finalLeaf, 0); err != nil {
		return false, err
	}
	committed := true
	syncErr := unix.Fsync(parent)
	unlinkErr := unix.Unlinkat(parent, stagedLeaf, 0)
	cleanupSyncErr := unix.Fsync(parent)
	return committed, errors.Join(syncErr, unlinkErr, cleanupSyncErr)
}

func writeExclusiveNoFollow(path string, data []byte, mode, directoryMode os.FileMode) error {
	parent, leaf, err := openParentNoFollow(path, true, directoryMode)
	if err != nil {
		return err
	}
	defer unix.Close(parent)
	fd, err := unix.Openat(parent, leaf, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, uint32(mode.Perm()))
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), path)
	ok := false
	defer func() {
		_ = file.Close()
		if !ok {
			_ = file.Sync()
		}
	}()
	if err := file.Chmod(mode.Perm()); err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	ok = true
	return unix.Fsync(parent)
}

func syncDirectoryNoFollow(path string) error {
	parent, leaf, err := openParentNoFollow(filepath.Join(path, ".sentinel"), false, 0)
	if err != nil {
		return err
	}
	defer unix.Close(parent)
	_ = leaf
	return unix.Fsync(parent)
}
