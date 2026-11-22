//go:build windows

package runner

import "os/exec"

// applyProcAttr is a no-op on Windows; kill uses taskkill /T (see internal/killtree).
// Detach/Job Object wiring is tracked separately; NOT verified on Windows in this PR.
func applyProcAttr(cmd *exec.Cmd) {
	// Intentionally empty: Windows SysProcAttr has no Setpgid.
}
