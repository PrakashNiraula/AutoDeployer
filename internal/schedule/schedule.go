// Package schedule manages data/schedules.json and OS task markers.
package schedule

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const fileName = "schedules.json"

type File struct {
	Version   int        `json:"version"`
	Schedules []Schedule `json:"schedules"`
}

type Schedule struct {
	ID         string `json:"id"`
	JobID      string `json:"job_id"`
	EnvProfile string `json:"env_profile"`
	Cron       string `json:"cron"`
	Enabled    bool   `json:"enabled"`
	OSTaskName string `json:"os_task_name"`
}

func Path(dataDir string) string {
	return filepath.Join(dataDir, fileName)
}

func Load(dataDir string) (File, error) {
	p := Path(dataDir)
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return File{Version: 1, Schedules: []Schedule{}}, nil
		}
		return File{}, err
	}
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return File{}, err
	}
	if f.Version == 0 {
		f.Version = 1
	}
	if f.Schedules == nil {
		f.Schedules = []Schedule{}
	}
	return f, nil
}

func Save(dataDir string, f File) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	if f.Version == 0 {
		f.Version = 1
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(dataDir), append(b, '\n'), 0o644)
}

func Set(dataDir string, s Schedule) error {
	if s.ID == "" || s.JobID == "" || s.EnvProfile == "" || s.Cron == "" {
		return fmt.Errorf("id, job_id, env_profile, cron required")
	}
	if s.OSTaskName == "" {
		s.OSTaskName = "AutoDeployer-" + s.ID
	}
	f, err := Load(dataDir)
	if err != nil {
		return err
	}
	found := false
	for i := range f.Schedules {
		if f.Schedules[i].ID == s.ID {
			f.Schedules[i] = s
			found = true
			break
		}
	}
	if !found {
		f.Schedules = append(f.Schedules, s)
	}
	if err := Save(dataDir, f); err != nil {
		return err
	}
	return InstallOS(dataDir, s)
}

func Delete(dataDir, id string) error {
	f, err := Load(dataDir)
	if err != nil {
		return err
	}
	out := f.Schedules[:0]
	var removed *Schedule
	for _, s := range f.Schedules {
		if s.ID == id {
			cp := s
			removed = &cp
			continue
		}
		out = append(out, s)
	}
	f.Schedules = out
	if err := Save(dataDir, f); err != nil {
		return err
	}
	if removed != nil {
		_ = UninstallOS(dataDir, *removed)
	}
	return nil
}

func MarkerDir(dataDir string) string {
	return filepath.Join(dataDir, "os-tasks")
}

func markerPath(dataDir, id string) string {
	return filepath.Join(MarkerDir(dataDir), id+".task")
}

// InstallOS writes a durable OS-task marker (and best-effort real installer).
func InstallOS(dataDir string, s Schedule) error {
	if err := os.MkdirAll(MarkerDir(dataDir), 0o755); err != nil {
		return err
	}
	line := cronLine(dataDir, s)
	body := fmt.Sprintf("os=%s\nname=%s\nenabled=%v\nline=%s\n", runtime.GOOS, s.OSTaskName, s.Enabled, line)
	if err := os.WriteFile(markerPath(dataDir, s.ID), []byte(body), 0o644); err != nil {
		return err
	}
	// Best-effort: on Unix, maintain data/crontab.fragment for operators to install.
	return writeFragment(dataDir)
}

func UninstallOS(dataDir string, s Schedule) error {
	_ = os.Remove(markerPath(dataDir, s.ID))
	return writeFragment(dataDir)
}

func cronLine(dataDir string, s Schedule) string {
	root := filepath.Dir(dataDir) // install root = parent of data/
	bin := filepath.Join(root, "app", "autodeployer")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	return fmt.Sprintf("%s %s --root %s run %s --env %s --trigger scheduled --yes",
		s.Cron, bin, root, s.JobID, s.EnvProfile)
}

func writeFragment(dataDir string) error {
	f, err := Load(dataDir)
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# Managed by AutoDeployer — install with: crontab data/crontab.fragment\n")
	for _, s := range f.Schedules {
		if !s.Enabled {
			continue
		}
		b.WriteString("# AutoDeployer:" + s.ID + "\n")
		b.WriteString(cronLine(dataDir, s) + "\n")
	}
	return os.WriteFile(filepath.Join(dataDir, "crontab.fragment"), []byte(b.String()), 0o644)
}

type Drift struct {
	ScheduleID string `json:"schedule_id"`
	Issue      string `json:"issue"`
}

// Reconcile compares schedules.json to OS markers.
func Reconcile(dataDir string) ([]Drift, error) {
	f, err := Load(dataDir)
	if err != nil {
		return nil, err
	}
	var drifts []Drift
	seen := map[string]bool{}
	for _, s := range f.Schedules {
		seen[s.ID] = true
		if _, err := os.Stat(markerPath(dataDir, s.ID)); err != nil {
			drifts = append(drifts, Drift{ScheduleID: s.ID, Issue: "missing_os_task"})
		}
	}
	entries, _ := os.ReadDir(MarkerDir(dataDir))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".task") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".task")
		if !seen[id] {
			drifts = append(drifts, Drift{ScheduleID: id, Issue: "orphan_os_task"})
		}
	}
	return drifts, nil
}

// OrphansAfterPromote lists schedules whose job_id no longer exists under project/jobs.
func OrphansAfterPromote(dataDir, jobsDir string) ([]Schedule, error) {
	f, err := Load(dataDir)
	if err != nil {
		return nil, err
	}
	var orphans []Schedule
	for _, s := range f.Schedules {
		if _, err := os.Stat(filepath.Join(jobsDir, s.JobID, "job.yaml")); err != nil {
			orphans = append(orphans, s)
		}
	}
	return orphans, nil
}
