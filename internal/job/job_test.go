package job

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	// internal/job -> repo root
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestSampleJobsValidate(t *testing.T) {
	root := repoRoot(t)
	jobs := filepath.Join(root, "project", "jobs")
	for _, id := range []string{"s3-pull-and-java", "db-backup"} {
		m, err := Find(jobs, id)
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		r := m.Validate()
		if !r.OK {
			t.Fatalf("%s: unexpected errors: %v", id, r.Errors)
		}
	}
}

func TestIDMustMatchFolder(t *testing.T) {
	dir := t.TempDir()
	jobDir := filepath.Join(dir, "myjob")
	_ = os.MkdirAll(filepath.Join(jobDir, "scripts"), 0o755)
	_ = os.WriteFile(filepath.Join(jobDir, "scripts", "run.bat"), []byte("@echo off\n"), 0o644)
	_ = os.WriteFile(filepath.Join(jobDir, "job.yaml"), []byte(`
schema_version: 1
id: wrong-id
name: X
workdir: scripts
steps:
  - name: a
    command: run.bat
required_env: []
`), 0o644)
	m, err := Load(jobDir)
	if err != nil {
		t.Fatal(err)
	}
	r := m.Validate()
	if r.OK {
		t.Fatal("expected id mismatch error")
	}
}

func TestSecretsSubsetRequiredEnv(t *testing.T) {
	dir := t.TempDir()
	jobDir := filepath.Join(dir, "j")
	_ = os.MkdirAll(filepath.Join(jobDir, "scripts"), 0o755)
	_ = os.WriteFile(filepath.Join(jobDir, "scripts", "run.bat"), []byte("x"), 0o644)
	_ = os.WriteFile(filepath.Join(jobDir, "job.yaml"), []byte(`
schema_version: 1
id: j
name: J
workdir: scripts
steps:
  - name: a
    command: run.bat
required_env: [A]
secrets: [SECRET_ONLY]
`), 0o644)
	m, err := Load(jobDir)
	if err != nil {
		t.Fatal(err)
	}
	r := m.Validate()
	if r.OK {
		t.Fatal("expected secrets subset error")
	}
}

func TestPathSafetyRejectsDotDot(t *testing.T) {
	dir := t.TempDir()
	jobDir := filepath.Join(dir, "safe")
	_ = os.MkdirAll(filepath.Join(jobDir, "scripts"), 0o755)
	_ = os.WriteFile(filepath.Join(jobDir, "scripts", "run.bat"), []byte("x"), 0o644)
	_ = os.WriteFile(filepath.Join(jobDir, "job.yaml"), []byte(`
schema_version: 1
id: safe
name: S
workdir: scripts
steps:
  - name: a
    command: ../outside.bat
required_env: []
`), 0o644)
	m, err := Load(jobDir)
	if err != nil {
		t.Fatal(err)
	}
	r := m.Validate()
	if r.OK {
		t.Fatal("expected path escape error")
	}
}

func TestDiscover(t *testing.T) {
	root := repoRoot(t)
	list, err := Discover(filepath.Join(root, "project", "jobs"))
	if err != nil {
		t.Fatal(err)
	}
	if len(list) < 2 {
		t.Fatalf("want at least 2 jobs, got %d", len(list))
	}
}
