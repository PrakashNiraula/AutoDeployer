package lockfile

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAcquireExclusive(t *testing.T) {
	dir := t.TempDir()
	data := filepath.Join(dir, "data")
	info, err := Acquire(data, "job-a", "run1", "tester")
	if err != nil {
		t.Fatal(err)
	}
	if info.PID != os.Getpid() {
		t.Fatalf("pid=%d", info.PID)
	}
	_, err = Acquire(data, "job-a", "run2", "tester")
	if err != ErrHeld {
		t.Fatalf("expected ErrHeld, got %v", err)
	}
	if err := Release(data, "job-a", "run1"); err != nil {
		t.Fatal(err)
	}
	if _, err := Acquire(data, "job-a", "run3", "tester"); err != nil {
		t.Fatal(err)
	}
}

func TestStalePID(t *testing.T) {
	dir := t.TempDir()
	data := filepath.Join(dir, "data")
	if err := os.MkdirAll(Dir(data), 0o755); err != nil {
		t.Fatal(err)
	}
	// Write a lock with a dead PID.
	p := Path(data, "job-b")
	body := `{
  "job_id": "job-b",
  "run_id": "old",
  "pid": 999999,
  "pid_start": "0",
  "started_at_utc": "2020-01-01T00:00:00Z",
  "started_local": "2020-01-01T00:00:00",
  "started_offset": "+00:00"
}`
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	held, _, err := IsHeld(data, "job-b")
	if err != nil {
		t.Fatal(err)
	}
	if held {
		t.Fatal("dead PID should be stale")
	}
	if _, err := Acquire(data, "job-b", "run-new", "t"); err != nil {
		t.Fatal(err)
	}
}

func TestWaitPromote(t *testing.T) {
	dir := t.TempDir()
	data := filepath.Join(dir, "data")
	if _, err := Acquire(data, PromoteName, "p1", "promoter"); err != nil {
		t.Fatal(err)
	}
	err := WaitPromote(data, 300*time.Millisecond)
	if err != ErrHeld {
		t.Fatalf("expected ErrHeld, got %v", err)
	}
	if err := Release(data, PromoteName, "p1"); err != nil {
		t.Fatal(err)
	}
	if err := WaitPromote(data, time.Second); err != nil {
		t.Fatal(err)
	}
}
