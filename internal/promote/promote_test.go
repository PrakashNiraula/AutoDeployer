package promote

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PrakashNiraula/autodeployer/internal/exitcode"
	"github.com/PrakashNiraula/autodeployer/internal/lockfile"
	"github.com/PrakashNiraula/autodeployer/internal/schedule"
	"github.com/PrakashNiraula/autodeployer/internal/store"
)

func TestPromoteHappyPath(t *testing.T) {
	root := t.TempDir()
	data := filepath.Join(root, "data")
	_ = os.MkdirAll(filepath.Join(root, "project", "jobs", "old"), 0o755)
	_ = os.WriteFile(filepath.Join(root, "project", "jobs", "old", "job.yaml"), []byte("schema_version: 1\nid: old\nname: Old\nworkdir: .\nsteps: [{name: n, command: x}]\n"), 0o644)
	src := filepath.Join(root, "incoming")
	_ = os.MkdirAll(filepath.Join(src, "jobs", "newjob"), 0o755)
	_ = os.WriteFile(filepath.Join(src, "jobs", "newjob", "job.yaml"), []byte("schema_version: 1\nid: newjob\nname: New\nworkdir: .\nsteps: [{name: n, command: x}]\n"), 0o644)
	_ = os.MkdirAll(data, 0o755)
	_ = schedule.Set(data, schedule.Schedule{ID: "orphan", JobID: "old", EnvProfile: "dev", Cron: "0 1 * * *", Enabled: true})

	res, err := Run(root, src, "tester")
	if err != nil || !res.OK {
		t.Fatalf("err=%v res=%+v", err, res)
	}
	if _, err := os.Stat(filepath.Join(root, "project", "jobs", "newjob", "job.yaml")); err != nil {
		t.Fatal(err)
	}
	if len(res.OrphanSchedules) != 1 || res.OrphanSchedules[0].JobID != "old" {
		t.Fatalf("orphans=%v", res.OrphanSchedules)
	}
	// data untouched marker
	if _, err := os.Stat(filepath.Join(data, "schedules.json")); err != nil {
		t.Fatal("data should remain")
	}
}

func TestPromoteRefusesActiveRun(t *testing.T) {
	root := t.TempDir()
	data := filepath.Join(root, "data")
	_ = os.MkdirAll(filepath.Join(root, "project", "jobs"), 0o755)
	src := filepath.Join(root, "incoming")
	_ = os.MkdirAll(filepath.Join(src, "jobs"), 0o755)
	db, err := store.Open(data)
	if err != nil {
		t.Fatal(err)
	}
	pid := os.Getpid()
	_ = db.InsertRun(store.Run{
		ID: "r1", ShortID: "r1", JobID: "j", EnvProfile: "dev", Trigger: "manual",
		Status: "running", StartedAtUTC: "x", StartedLocal: "x", StartedOffset: "+00:00", PID: &pid,
	})
	_ = db.Close()
	res, err := Run(root, src, "t")
	if res.CLIExit != exitcode.PromoteFailed {
		t.Fatalf("cli=%d err=%v", res.CLIExit, err)
	}
}

func TestPromoteLockBlocks(t *testing.T) {
	root := t.TempDir()
	data := filepath.Join(root, "data")
	src := filepath.Join(root, "incoming")
	_ = os.MkdirAll(filepath.Join(src, "jobs"), 0o755)
	if _, err := lockfile.Acquire(data, lockfile.PromoteName, "other", "x"); err != nil {
		t.Fatal(err)
	}
	res, err := Run(root, src, "t")
	if res.CLIExit != exitcode.PromoteLockHeld {
		t.Fatalf("cli=%d err=%v", res.CLIExit, err)
	}
}
