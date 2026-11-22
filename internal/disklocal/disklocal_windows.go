//go:build windows

package disklocal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Check reports whether path lives on a local disk (rejects UNC).
func Check(path string) (local bool, fstype string, err error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false, "", err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return false, "", err
	}
	if strings.HasPrefix(abs, `\\`) || strings.HasPrefix(abs, "//") {
		return false, "unc", nil
	}
	return true, "local", nil
}

func EnsureLocal(path string) error {
	ok, fstype, err := Check(path)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("data/ appears to be on remote filesystem %q; must be local disk", fstype)
	}
	return nil
}
