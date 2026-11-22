package gc

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLogsRetention(t *testing.T) {
	dir := t.TempDir()
	data := filepath.Join(dir, "data")
	logDir := filepath.Join(data, "logs", "j")
	_ = os.MkdirAll(logDir, 0o755)
	old := filepath.Join(logDir, "old.log")
	newf := filepath.Join(logDir, "new.log")
	_ = os.WriteFile(old, []byte("x"), 0o644)
	_ = os.WriteFile(newf, []byte("y"), 0o644)
	past := time.Now().Add(-48 * time.Hour)
	_ = os.Chtimes(old, past, past)
	res, err := Logs(data, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Deleted) != 1 {
		t.Fatalf("%+v", res)
	}
	if _, err := os.Stat(newf); err != nil {
		t.Fatal("new should remain")
	}
}
