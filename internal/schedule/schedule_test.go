package schedule

import (
	"path/filepath"
	"testing"
)

func TestSetReconcileDelete(t *testing.T) {
	dir := t.TempDir()
	data := filepath.Join(dir, "data")
	_ = Set(data, Schedule{ID: "nightly", JobID: "demo", EnvProfile: "dev", Cron: "0 2 * * *", Enabled: true})
	f, err := Load(data)
	if err != nil || len(f.Schedules) != 1 {
		t.Fatalf("%v %#v", err, f)
	}
	drifts, err := Reconcile(data)
	if err != nil || len(drifts) != 0 {
		t.Fatalf("drifts=%v err=%v", drifts, err)
	}
	if err := Delete(data, "nightly"); err != nil {
		t.Fatal(err)
	}
	drifts, _ = Reconcile(data)
	// orphan marker may remain cleared by Delete
	for _, d := range drifts {
		if d.Issue == "missing_os_task" {
			t.Fatalf("%v", drifts)
		}
	}
}
