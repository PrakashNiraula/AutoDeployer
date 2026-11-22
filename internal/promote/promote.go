// Package promote replaces project/ safely without touching data/.
package promote

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/PrakashNiraula/autodeployer/internal/adtime"
	"github.com/PrakashNiraula/autodeployer/internal/exitcode"
	"github.com/PrakashNiraula/autodeployer/internal/lockfile"
	"github.com/PrakashNiraula/autodeployer/internal/schedule"
	"github.com/PrakashNiraula/autodeployer/internal/store"
)

type Result struct {
	OK              bool                `json:"ok"`
	OldVersion      string              `json:"old_version"`
	NewVersion      string              `json:"new_version"`
	DeploymentID    string              `json:"deployment_id,omitempty"`
	OrphanSchedules []schedule.Schedule `json:"orphan_schedules"`
	CLIExit         int                 `json:"-"`
}

// Run promotes sourceDir into root/project.
func Run(root, sourceDir, actor string) (Result, error) {
	dataDir := filepath.Join(root, "data")
	project := filepath.Join(root, "project")
	absSrc, err := filepath.Abs(sourceDir)
	if err != nil {
		return Result{CLIExit: exitcode.Usage}, err
	}
	if _, err := os.Stat(filepath.Join(absSrc, "jobs")); err != nil {
		return Result{CLIExit: exitcode.Usage}, fmt.Errorf("source must contain jobs/: %w", err)
	}

	if _, err := lockfile.Acquire(dataDir, lockfile.PromoteName, "promote", actor); err != nil {
		return Result{CLIExit: exitcode.PromoteLockHeld}, err
	}
	defer func() { _ = lockfile.Release(dataDir, lockfile.PromoteName, "promote") }()

	db, err := store.Open(dataDir)
	if err != nil {
		return Result{CLIExit: exitcode.Internal}, err
	}
	defer db.Close()
	_, _ = db.ReconcileStaleRunning()
	active, err := db.ActiveRuns("")
	if err != nil {
		return Result{CLIExit: exitcode.Internal}, err
	}
	if len(active) > 0 {
		return Result{CLIExit: exitcode.PromoteFailed}, fmt.Errorf("active runs present")
	}

	oldVer := readVersion(dataDir)
	newVer, err := hashTree(absSrc)
	if err != nil {
		return Result{CLIExit: exitcode.Internal}, err
	}

	backup := project + ".bak." + time.Now().Format("20060102T150405")
	staging := project + ".new." + time.Now().Format("20060102T150405")

	if err := copyDir(absSrc, staging); err != nil {
		_ = os.RemoveAll(staging)
		return Result{CLIExit: exitcode.Internal}, err
	}

	if err := renameRetry(project, backup); err != nil && !os.IsNotExist(err) {
		_ = os.RemoveAll(staging)
		return Result{CLIExit: exitcode.Internal}, fmt.Errorf("backup rename: %w", err)
	}
	if err := renameRetry(staging, project); err != nil {
		_ = renameRetry(backup, project) // rollback
		_ = os.RemoveAll(staging)
		return Result{CLIExit: exitcode.Internal}, fmt.Errorf("promote rename: %w", err)
	}
	_ = os.RemoveAll(backup)

	inst := adtime.Now()
	if err := writeVersion(dataDir, newVer, absSrc, inst); err != nil {
		return Result{CLIExit: exitcode.Internal}, err
	}
	depID, _ := store.NewRunID()
	if err := db.InsertDeployment(depID, oldVer, newVer, absSrc, actor, "succeeded", inst); err != nil {
		return Result{CLIExit: exitcode.Internal}, err
	}

	orphans, _ := schedule.OrphansAfterPromote(dataDir, filepath.Join(project, "jobs"))
	return Result{
		OK: true, OldVersion: oldVer, NewVersion: newVer, DeploymentID: depID,
		OrphanSchedules: orphans, CLIExit: exitcode.OK,
	}, nil
}

func renameRetry(oldpath, newpath string) error {
	var err error
	for i := 0; i < 8; i++ {
		err = os.Rename(oldpath, newpath)
		if err == nil {
			return nil
		}
		time.Sleep(time.Duration(50*(i+1)) * time.Millisecond)
	}
	return err
}

func hashTree(root string) (string, error) {
	h := sha256.New()
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		h.Write([]byte(rel))
		h.Write([]byte{0})
		if info.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := io.Copy(h, f); err != nil {
			return err
		}
		h.Write([]byte{0})
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func readVersion(dataDir string) string {
	b, err := os.ReadFile(filepath.Join(dataDir, "project-version.json"))
	if err != nil {
		return "unversioned"
	}
	var v struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(b, &v) != nil || v.Version == "" {
		return "unversioned"
	}
	return v.Version
}

func writeVersion(dataDir, version, source string, inst adtime.Instant) error {
	payload := map[string]string{
		"version":     version,
		"computed_at": inst.Canonical,
		"source_path": source,
	}
	b, _ := json.MarshalIndent(payload, "", "  ")
	return os.WriteFile(filepath.Join(dataDir, "project-version.json"), append(b, '\n'), 0o644)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}
