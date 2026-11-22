package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/PrakashNiraula/autodeployer/internal/adtime"
	"github.com/PrakashNiraula/autodeployer/internal/lockfile"
)

func TestNewRunIDUnique(t *testing.T) {
	const n = 10000
	seenID := make(map[string]struct{}, n)
	seenShort := make(map[string]struct{}, n)
	var mu sync.Mutex
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, short := NewRunID()
			if len(id) != 32 {
				errs <- fmt.Errorf("id len=%d", len(id))
				return
			}
			if len(short) != 8 || short != id[:8] {
				errs <- fmt.Errorf("short=%q id=%q", short, id)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			if _, ok := seenID[id]; ok {
				errs <- fmt.Errorf("duplicate id %s", id)
				return
			}
			seenID[id] = struct{}{}
			// short_id may collide in theory (48 bits); for 10k samples expect uniqueness
			if _, ok := seenShort[short]; ok {
				errs <- fmt.Errorf("duplicate short_id %s", short)
				return
			}
			seenShort[short] = struct{}{}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}

func TestTwoWritersWAL(t *testing.T) {
	dir := t.TempDir()
	data := filepath.Join(dir, "data")
	db, err := Open(data)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, short := NewRunID()
			inst := adtime.Now()
			r := Run{
				ID: id, ShortID: short, JobID: "j", EnvProfile: "dev",
				Trigger: "manual", Status: "succeeded",
				StartedAtUTC: inst.UTC, StartedLocal: inst.Local, StartedOffset: inst.Offset,
			}
			if err := db.InsertRun(r); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	list, err := db.ListRuns("", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 10 {
		t.Fatalf("got %d runs", len(list))
	}
}

func TestReconcileStaleRunning(t *testing.T) {
	dir := t.TempDir()
	data := filepath.Join(dir, "data")
	db, err := Open(data)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	inst := adtime.Now()
	dead := 999999
	r := Run{
		ID: "run-dead", ShortID: "dead0001", JobID: "j", EnvProfile: "dev",
		Trigger: "manual", Status: "running", PID: &dead,
		StartedAtUTC: inst.UTC, StartedLocal: inst.Local, StartedOffset: inst.Offset,
	}
	if err := db.InsertRun(r); err != nil {
		t.Fatal(err)
	}
	fixed, err := db.ReconcileStaleRunning()
	if err != nil {
		t.Fatal(err)
	}
	if len(fixed) != 1 || fixed[0] != "run-dead" {
		t.Fatalf("fixed=%v", fixed)
	}
	got, err := db.GetRun("run-dead")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "failed" {
		t.Fatalf("status=%s", got.Status)
	}
}

func TestEnvAuditNoValue(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "data"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	inst := adtime.Now()
	if err := db.InsertEnvAudit("a1", "me", "job", "dev", "PASSWORD", inst); err != nil {
		t.Fatal(err)
	}
	row := db.sql.QueryRow(`SELECT key FROM env_audit WHERE id=?`, "a1")
	var key string
	if err := row.Scan(&key); err != nil {
		t.Fatal(err)
	}
	if key != "PASSWORD" {
		t.Fatal(key)
	}
}

func TestReconcileAllClearsLocks(t *testing.T) {
	dir := t.TempDir()
	data := filepath.Join(dir, "data")
	db, err := Open(data)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := os.MkdirAll(lockfile.Dir(data), 0o755); err != nil {
		t.Fatal(err)
	}
	p := lockfile.Path(data, "x")
	body := `{"job_id":"x","pid":999999,"pid_start":"0","started_at_utc":"a","started_local":"a","started_offset":"+00:00"}`
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	fix, err := ReconcileAll(data, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(fix.ClearedLocks) != 1 {
		t.Fatalf("cleared=%v", fix.ClearedLocks)
	}
}
