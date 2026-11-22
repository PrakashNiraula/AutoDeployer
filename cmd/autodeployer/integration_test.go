package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestIntegrationSampleJob(t *testing.T) {
	root := buildSampleRoot(t)
	code := run([]string{"--root", root, "doctor", "--json"})
	if code != 0 {
		t.Fatalf("doctor=%d", code)
	}
	code = run([]string{"--root", root, "schema", "validate", "all"})
	if code != 0 {
		t.Fatalf("schema=%d", code)
	}
	code = run([]string{"--root", root, "jobs", "list"})
	if code != 0 {
		t.Fatalf("jobs=%d", code)
	}

	// Capture plan_id via plan package path: run plan and parse stdout by reusing Build is heavier;
	// invoke plan command and read from a temp by wrapping print — instead call twice with mismatch first.
	code = run([]string{"--root", root, "run", "demo", "--env", "dev", "--trigger", "manual", "--plan-id", "bad", "--yes", "--quiet"})
	if code != 11 {
		t.Fatalf("want plan mismatch 11, got %d", code)
	}

	planID := computePlanID(t, root)
	meta := filepath.Join(root, "meta.json")
	code = run([]string{"--root", root, "run", "demo", "--env", "dev", "--trigger", "manual", "--plan-id", planID, "--yes", "--quiet", "--json-meta", meta})
	if code != 0 {
		t.Fatalf("run=%d", code)
	}
	b, err := os.ReadFile(meta)
	if err != nil {
		t.Fatal(err)
	}
	var res map[string]any
	if err := json.Unmarshal(b, &res); err != nil {
		t.Fatal(err)
	}
	if res["status"] != "succeeded" {
		t.Fatalf("%v", res)
	}
	runID, _ := res["run_id"].(string)
	code = run([]string{"--root", root, "history", "show", runID})
	if code != 0 {
		t.Fatalf("history show=%d", code)
	}
	code = run([]string{"--root", root, "logs", "tail", runID})
	if code != 0 {
		t.Fatalf("logs tail=%d", code)
	}
	code = run([]string{"--root", root, "cancel", "missing-run"})
	if code != 13 {
		t.Fatalf("cancel missing want 13 got %d", code)
	}
}

func computePlanID(t *testing.T, root string) string {
	t.Helper()
	// Use plan --json by capturing via temporary stdout redirect.
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	code := run([]string{"--root", root, "plan", "demo", "--env", "dev", "--json"})
	_ = w.Close()
	os.Stdout = old
	buf := make([]byte, 1<<20)
	n, _ := r.Read(buf)
	_ = r.Close()
	if code != 0 {
		t.Fatalf("plan=%d %s", code, buf[:n])
	}
	var m map[string]any
	if err := json.Unmarshal(buf[:n], &m); err != nil {
		t.Fatalf("json: %v %s", err, buf[:n])
	}
	id, _ := m["plan_id"].(string)
	if id == "" {
		t.Fatal("empty plan_id")
	}
	return id
}

func buildSampleRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	jobDir := filepath.Join(root, "project", "jobs", "demo")
	scripts := filepath.Join(jobDir, "scripts")
	_ = os.MkdirAll(scripts, 0o755)
	cmd := "echo_ok.sh"
	body := "#!/bin/sh\necho ok\nexit 0\n"
	if runtime.GOOS == "windows" {
		cmd = "echo_ok.bat"
		body = "@echo off\r\necho ok\r\nexit /b 0\r\n"
	}
	_ = os.WriteFile(filepath.Join(jobDir, "job.yaml"), []byte(`schema_version: 1
id: demo
name: Demo
workdir: scripts
shell: auto
encoding:
  output_code_page: utf-8
steps:
  - name: Echo
    command: `+cmd+`
required_env: [TOKEN]
secrets: [TOKEN]
timeout_minutes: 1
`), 0o644)
	_ = os.WriteFile(filepath.Join(scripts, cmd), []byte(body), 0o755)
	data := filepath.Join(root, "data")
	_ = os.MkdirAll(filepath.Join(data, "env", "demo"), 0o755)
	_ = os.WriteFile(filepath.Join(data, "env", "demo", ".env.dev"), []byte("TOKEN=secretpw\n"), 0o600)
	_ = os.WriteFile(filepath.Join(data, "machine.yaml"), []byte("machine_name: ci\nallowed_envs: [dev]\ndefault_env: dev\n"), 0o644)
	return root
}
