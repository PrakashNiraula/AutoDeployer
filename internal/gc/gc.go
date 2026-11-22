// Package gc applies log retention under data/logs.
package gc

import (
	"os"
	"path/filepath"
	"time"
)

type Result struct {
	Deleted []string `json:"deleted"`
	Kept    int      `json:"kept"`
}

// Logs deletes log files older than retentionDays. Never touches project/.
func Logs(dataDir string, retentionDays int) (Result, error) {
	res := Result{Deleted: []string{}}
	if retentionDays <= 0 {
		retentionDays = 30
	}
	cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)
	root := filepath.Join(dataDir, "logs")
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(path); err == nil {
				res.Deleted = append(res.Deleted, path)
			}
		} else {
			res.Kept++
		}
		return nil
	})
	return res, nil
}
