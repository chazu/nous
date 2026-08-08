//go:build !darwin && !linux

package causalexpv2

import (
	"errors"
	"runtime"
)

// Unsupported platforms fail closed rather than weakening no-replace
// publication to a racy check followed by rename.
func renameNoReplace(_, _ string) error {
	return errors.New("atomic no-replace directory rename is unsupported on " + runtime.GOOS)
}
