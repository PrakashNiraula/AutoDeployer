// Package root resolves the AutoDeployer install root (--root or discovery).
package root

import (
	"fmt"
	"os"
	"path/filepath"
)

// Resolve returns an absolute install root containing project/ (and preferably data/).
func Resolve(flagRoot string) (string, error) {
	if flagRoot != "" {
		abs, err := filepath.Abs(flagRoot)
		if err != nil {
			return "", fmt.Errorf("root %q: %w", flagRoot, err)
		}
		if err := CheckLayout(abs); err != nil {
			return "", err
		}
		return abs, nil
	}
	candidates := []string{}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, wd)
		candidates = append(candidates, filepath.Dir(wd))
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Dir(exe))
		candidates = append(candidates, filepath.Dir(filepath.Dir(exe)))
	}
	seen := map[string]bool{}
	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err != nil || seen[abs] {
			continue
		}
		seen[abs] = true
		if CheckLayout(abs) == nil {
			return abs, nil
		}
	}
	return "", fmt.Errorf("could not find AutoDeployer root (need a folder with project/); pass --root")
}

// CheckLayout ensures project/ exists under root.
func CheckLayout(root string) error {
	proj := filepath.Join(root, "project")
	st, err := os.Stat(proj)
	if err != nil {
		return fmt.Errorf("root %s: missing project/ folder", root)
	}
	if !st.IsDir() {
		return fmt.Errorf("root %s: project is not a directory", root)
	}
	return nil
}

func ProjectDir(root string) string { return filepath.Join(root, "project") }
func DataDir(root string) string    { return filepath.Join(root, "data") }
func JobsDir(root string) string    { return filepath.Join(root, "project", "jobs") }
