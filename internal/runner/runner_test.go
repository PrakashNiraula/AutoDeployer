package runner

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/PrakashNiraula/autodeployer/internal/exitcode"
	"github.com/PrakashNiraula/autodeployer/internal/job"
	"github.com/PrakashNiraula/autodeployer/internal/machine"
	"github.com/PrakashNiraula/autodeployer/internal/plan"
)

func TestPlanIDMismatch(t *testing.T) {
	root := setupFixture(t, fixtureOpts{})
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
	root := setupFixture(t, fixtureOpts{japanese: true})
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
	log := string(b)
	if strings.Contains(log, "secretpw") {
		t.Fatalf("secret not masked in log: %s", log)
	}
	if !strings.Contains(log, "***") {
		t.Fatalf("expected masked secret marker in log: %s", log)
	}
	if !strings.Contains(log, "日本語") {
		t.Fatalf("expected Japanese text in log: %s", log)
	}
}

func TestRunNonZeroExit(t *testing.T) {
	root := setupFixture(t, fixtureOpts{fail: true})
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
	res, _ := Execute(Options{
		Root: root, Job: m, EnvProfile: "dev", Trigger: "manual",
		PlanID: pl.PlanID, Yes: true, Quiet: true,
	})
	if res.CLIExit != exitcode.JobFailed {
		t.Fatalf("cli=%d status=%s want JobFailed", res.CLIExit, res.Status)
	}
	if res.Status != "failed" {
		t.Fatalf("status=%s", res.Status)
	}
}

func TestNestedBatWithoutCallDocumented(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows cmd.exe nested bat only")
	}
	root := t.TempDir()
	jobDir := filepath.Join(root, "project", "jobs", "nested")
	scripts := filepath.Join(jobDir, "scripts")
	_ = os.MkdirAll(scripts, 0o755)
	_ = os.WriteFile(filepath.Join(jobDir, "job.yaml"), []byte(`schema_version: 1
id: nested
name: Nested
workdir: scripts
shell: auto
encoding:
  output_code_page: utf-8
steps:
  - name: Outer
    command: outer.bat
required_env: [TOKEN]
secrets: [TOKEN]
timeout_minutes: 1
`), 0o644)
	// outer.bat invokes inner.bat WITHOUT call — classic cmd exit-code pitfall
	_ = os.WriteFile(filepath.Join(scripts, "outer.bat"), []byte("@echo off\r\ninner.bat\r\nexit /b 0\r\n"), 0o644)
	_ = os.WriteFile(filepath.Join(scripts, "inner.bat"), []byte("@echo off\r\nexit /b 42\r\n"), 0o644)
	data := filepath.Join(root, "data")
	_ = os.MkdirAll(filepath.Join(data, "env", "nested"), 0o755)
	_ = os.WriteFile(filepath.Join(data, "env", "nested", ".env.dev"), []byte("TOKEN=secretpw\n"), 0o600)
	_ = os.WriteFile(filepath.Join(data, "machine.yaml"), []byte("machine_name: t\nallowed_envs: [dev]\ndefault_env: dev\n"), 0o644)

	m, err := job.Find(filepath.Join(root, "project", "jobs"), "nested")
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ := machine.Load(data)
	pl, err := plan.Build(root, m, "dev", cfg, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, _ := Execute(Options{
		Root: root, Job: m, EnvProfile: "dev", Trigger: "manual",
		PlanID: pl.PlanID, Yes: true, Quiet: true,
	})
	// Document observed exit: without `call`, cmd often reports 0 from the outer script.
	t.Logf("observed nested-without-call: status=%s cli=%d step_exit=%v (documented; lint should warn)", res.Status, res.CLIExit, res.Steps)
}

type fixtureOpts struct {
	fail     bool
	japanese bool
}

func setupFixture(t *testing.T, opts fixtureOpts) string {
	t.Helper()
	root := t.TempDir()
	jobDir := filepath.Join(root, "project", "jobs", "demo")
	scripts := filepath.Join(jobDir, "scripts")
	if err := os.MkdirAll(scripts, 0o755); err != nil {
		t.Fatal(err)
	}

	cmdName := "echo_ok.sh"
	script := "#!/bin/sh\necho hello\necho token=${TOKEN}\n"
	if opts.japanese {
		script += "echo 日本語テスト\n"
	}
	if opts.fail {
		script += "exit 7\n"
	} else {
		script += "exit 0\n"
	}
	if runtime.GOOS == "windows" {
		cmdName = "echo_ok.bat"
		script = "@echo off\r\necho hello\r\necho token=%TOKEN%\r\n"
		if opts.japanese {
			script += "echo 日本語テスト\r\n"
		}
		if opts.fail {
			script += "exit /b 7\r\n"
		} else {
			script += "exit /b 0\r\n"
		}
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
    command: ` + cmdName + `
required_env: [TOKEN]
secrets: [TOKEN]
timeout_minutes: 1
allow_concurrent: false
`
	if err := os.WriteFile(filepath.Join(jobDir, "job.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scripts, cmdName), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(root, "data")
	_ = os.MkdirAll(filepath.Join(data, "env", "demo"), 0o755)
	_ = os.WriteFile(filepath.Join(data, "env", "demo", ".env.dev"), []byte("TOKEN=secretpw\n"), 0o600)
	_ = os.WriteFile(filepath.Join(data, "machine.yaml"), []byte("machine_name: t\nallowed_envs: [dev]\ndefault_env: dev\n"), 0o644)
	return root
}
