// Package lockfile implements exclusive job and promote locks under data/locks/.
package lockfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/PrakashNiraula/autodeployer/internal/adtime"
	"github.com/PrakashNiraula/autodeployer/internal/procinfo"
)

const PromoteName = "_promote"

// Info is the JSON body of a lock file.
type Info struct {
	JobID         string `json:"job_id"`
	RunID         string `json:"run_id,omitempty"`
	PID           int    `json:"pid"`
	PIDStart      string `json:"pid_start"`
	StartedAtUTC  string `json:"started_at_utc"`
	StartedLocal  string `json:"started_local"`
	StartedOffset string `json:"started_offset"`
	Actor         string `json:"actor,omitempty"`
}

var (
	ErrHeld  = errors.New("lock held")
	ErrStale = errors.New("lock stale")
)

func Dir(dataDir string) string {
	return filepath.Join(dataDir, "locks")
}

func Path(dataDir, jobID string) string {
	return filepath.Join(Dir(dataDir), jobID+".lock")
}

func PromotePath(dataDir string) string {
	return Path(dataDir, PromoteName)
}

// Acquire creates an exclusive lock file. Returns ErrHeld if a live lock exists.
func Acquire(dataDir, jobID, runID, actor string) (Info, error) {
	if err := os.MkdirAll(Dir(dataDir), 0o755); err != nil {
		return Info{}, err
	}
	p := Path(dataDir, jobID)
	if err := clearIfStale(p); err != nil && !errors.Is(err, ErrStale) && !os.IsNotExist(err) {
		if errors.Is(err, ErrHeld) {
			return Info{}, ErrHeld
		}
	}
	pid := os.Getpid()
	start, err := procinfo.StartKey(pid)
	if err != nil {
		return Info{}, err
	}
	inst := adtime.Now()
	info := Info{
		JobID:         jobID,
		RunID:         runID,
		PID:           pid,
		PIDStart:      start,
		StartedAtUTC:  inst.UTC,
		StartedLocal:  inst.Local,
		StartedOffset: inst.Offset,
		Actor:         actor,
	}
	b, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return Info{}, err
	}
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			if clearIfStale(p) == nil {
				f, err = os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
			}
			if err != nil {
				return Info{}, ErrHeld
			}
		} else {
			return Info{}, err
		}
	}
	defer f.Close()
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		_ = os.Remove(p)
		return Info{}, err
	}
	return info, nil
}

// Release removes the lock if it matches runID (or runID empty).
func Release(dataDir, jobID, runID string) error {
	p := Path(dataDir, jobID)
	info, err := Read(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if runID != "" && info.RunID != "" && info.RunID != runID {
		return fmt.Errorf("lock owned by different run")
	}
	return os.Remove(p)
}

// Read parses a lock file.
func Read(path string) (Info, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Info{}, err
	}
	var info Info
	if err := json.Unmarshal(b, &info); err != nil {
		return Info{}, err
	}
	return info, nil
}

// IsHeld reports whether a live (non-stale) lock exists.
func IsHeld(dataDir, jobID string) (bool, Info, error) {
	p := Path(dataDir, jobID)
	info, err := Read(p)
	if err != nil {
		if os.IsNotExist(err) {
			return false, Info{}, nil
		}
		return false, Info{}, err
	}
	if stale(info) {
		return false, info, nil
	}
	return true, info, nil
}

// WaitPromote waits until _promote.lock is clear or timeout. Returns ErrHeld if still held.
func WaitPromote(dataDir string, wait time.Duration) error {
	deadline := time.Now().Add(wait)
	for {
		held, _, err := IsHeld(dataDir, PromoteName)
		if err != nil {
			return err
		}
		if !held {
			_ = clearIfStale(PromotePath(dataDir))
			return nil
		}
		if time.Now().After(deadline) {
			return ErrHeld
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// ListStale returns paths of stale lock files under data/locks.
func ListStale(dataDir string) ([]string, error) {
	dir := Dir(dataDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".lock" {
			continue
		}
		p := filepath.Join(dir, e.Name())
		info, err := Read(p)
		if err != nil {
			out = append(out, p)
			continue
		}
		if stale(info) {
			out = append(out, p)
		}
	}
	return out, nil
}

// ClearStale removes stale locks; returns cleared paths.
func ClearStale(dataDir string) ([]string, error) {
	stalePaths, err := ListStale(dataDir)
	if err != nil {
		return nil, err
	}
	var cleared []string
	for _, p := range stalePaths {
		if err := os.Remove(p); err == nil {
			cleared = append(cleared, p)
		}
	}
	return cleared, nil
}

func clearIfStale(path string) error {
	info, err := Read(path)
	if err != nil {
		return err
	}
	if !stale(info) {
		return ErrHeld
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	return ErrStale
}

func stale(info Info) bool {
	alive, key, err := procinfo.Alive(info.PID)
	if err != nil || !alive {
		return true
	}
	if info.PIDStart != "" && key != "" && key != info.PIDStart {
		return true
	}
	return false
}
