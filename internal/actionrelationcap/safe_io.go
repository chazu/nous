package actionrelationcap

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/sys/unix"
)

func openParentNoFollow(path string) (int, string, error) {
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
		unix.Close(fd)
		if openErr != nil {
			return -1, "", openErr
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

func readRegularNoFollow(path string, mode os.FileMode, size int64) ([]byte, error) {
	parent, leaf, err := openParentNoFollow(path)
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
	if err := unix.Fstat(fd, &stat); err != nil || stat.Mode&unix.S_IFMT != unix.S_IFREG || os.FileMode(stat.Mode).Perm() != mode.Perm() || stat.Nlink != 1 || size >= 0 && stat.Size != size {
		return nil, fmt.Errorf("protected path is not an exclusive mode-%04o regular file", mode.Perm())
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func requireAbsentNoFollow(path string) error {
	parent, leaf, err := openParentNoFollow(path)
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
	return fmt.Errorf("protected path already exists: %s", path)
}

func readSecretNoFollow(path string) ([]byte, error) {
	return readRegularNoFollow(path, 0o600, 32)
}

func eraseSecretFile(path string) error {
	parent, leaf, err := openParentNoFollow(path)
	if err != nil {
		return err
	}
	defer unix.Close(parent)
	fd, err := unix.Openat(parent, leaf, unix.O_WRONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("open locked secret for erasure: %w", err)
	}
	file := os.NewFile(uintptr(fd), path)
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil || stat.Mode&unix.S_IFMT != unix.S_IFREG || os.FileMode(stat.Mode).Perm() != 0o600 || stat.Nlink != 1 || stat.Size != 32 {
		_ = file.Close()
		return fmt.Errorf("locked secret changed before erasure")
	}
	zeros := make([]byte, 32)
	_, writeErr := file.WriteAt(zeros, 0)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err := unix.Unlinkat(parent, leaf, 0); err != nil {
		return fmt.Errorf("delete erased locked secret: %w", err)
	}
	return unix.Fsync(parent)
}
