package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ResolveCommand picks the executable argv for a step command relative to workdir.
func ResolveCommand(workdir, command string) (argv []string, err error) {
	cmdPath := command
	if !filepath.IsAbs(command) {
		cmdPath = filepath.Join(workdir, command)
	}
	if runtime.GOOS == "windows" {
		lower := strings.ToLower(command)
		if strings.HasSuffix(lower, ".bat") || strings.HasSuffix(lower, ".cmd") {
			return []string{"cmd", "/C", "call", cmdPath}, nil
		}
		return []string{cmdPath}, nil
	}
	// Unix: prefer .sh sibling for .bat stubs so samples run in CI.
	if strings.HasSuffix(strings.ToLower(command), ".bat") {
		sh := strings.TrimSuffix(cmdPath, filepath.Ext(cmdPath)) + ".sh"
		if st, err := os.Stat(sh); err == nil && !st.IsDir() {
			return []string{"/bin/sh", sh}, nil
		}
		return nil, fmt.Errorf("unix: no .sh sibling for %s (add scripts/%s.sh)", command, strings.TrimSuffix(filepath.Base(command), filepath.Ext(command)))
	}
	if strings.HasSuffix(strings.ToLower(command), ".sh") {
		return []string{"/bin/sh", cmdPath}, nil
	}
	return []string{cmdPath}, nil
}
