package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/PrakashNiraula/autodeployer/internal/exitcode"
	"github.com/PrakashNiraula/autodeployer/internal/job"
	"github.com/PrakashNiraula/autodeployer/internal/killtree"
	"github.com/PrakashNiraula/autodeployer/internal/machine"
	"github.com/PrakashNiraula/autodeployer/internal/plan"
)

func TestPlanIDMismatch(t *testing.T) {
	root := setupFixture(t)
	m, err := job.Find(filepath.Join(root, "project", "jobs"), "demo")
	if err != nil {
		t.Fatal(err)
	}
	res, err := Execute(Options{
		Root: root, Job: m, EnvProfile: "dev", Trigger: "manual",
		PlanID: "deadbeef", Yes: true, Quiet: true,
	})
	if res.CLIExit != exitcode.PlanMismatch {
		t.Fatalf("cli=%d err=%v", res.CLIExit, err)
	}
}

func TestRunSucceeds(t *testing.T) {
	root := setupFixture(t)
	m, err := job.Find(filepath.Join(root, "project", "jobs"), "demo")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := machine.Load(filepath.Join(root, "data"))
	if err != nil {
		t.Fatal(err)
	}
	pl, err := plan.Build(root, m, "dev", cfg, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	meta := filepath.Join(root, "meta.json")
	res, err := Execute(Options{
		Root: root, Job: m, EnvProfile: "dev", Trigger: "manual",
		PlanID: pl.PlanID, Yes: true, Quiet: true, JSONMeta: meta,
	})
	if res.CLIExit != 0 {
		t.Fatalf("err=%v cli=%d status=%s", err, res.CLIExit, res.Status)
	}
	if res.Status != "succeeded" {
		t.Fatalf("status=%s", res.Status)
	}
	if _, err := os.Stat(res.LogPath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(meta); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(res.LogPath)
	if !contains(string(b), "***") && contains(string(b), "secretpw") {
		t.Fatalf("secret not masked in log: %s", b)
	}
}

func TestKillProcessGroup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix only")
	}
	cmd := exec.Command("sleep", "30")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	time.Sleep(50 * time.Millisecond)
	if err := killtree.KillProcessGroup(pid); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
}

func setupFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	jobDir := filepath.Join(root, "project", "jobs", "demo")
	scripts := filepath.Join(jobDir, "scripts")
	if err := os.MkdirAll(scripts, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := `schema_version: 1
id: demo
name: Demo
workdir: scripts
shell: auto
encoding:
  output_code_page: utf-8
steps:
  - name: Echo
    command: echo_ok.sh
required_env: [TOKEN]
secrets: [TOKEN]
timeout_minutes: 1
allow_concurrent: false
`
	if err := os.WriteFile(filepath.Join(jobDir, "job.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	sh := "#!/bin/sh\necho hello\necho token=${TOKEN}\nexit 0\n"
	if err := os.WriteFile(filepath.Join(scripts, "echo_ok.sh"), []byte(sh), 0o755); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(root, "data")
	_ = os.MkdirAll(filepath.Join(data, "env", "demo"), 0o755)
	_ = os.WriteFile(filepath.Join(data, "env", "demo", ".env.dev"), []byte("TOKEN=secretpw\n"), 0o600)
	_ = os.WriteFile(filepath.Join(data, "machine.yaml"), []byte("machine_name: t\nallowed_envs: [dev]\ndefault_env: dev\n"), 0o644)
	return root
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})())
}
