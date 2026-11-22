// Package store is the SQLite history database (WAL, pure-Go driver).
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PrakashNiraula/autodeployer/internal/adtime"
	"github.com/PrakashNiraula/autodeployer/internal/lockfile"
	"github.com/PrakashNiraula/autodeployer/internal/procinfo"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

const schemaVersion = "1"

const ddl = `
CREATE TABLE IF NOT EXISTS runs (
  id TEXT PRIMARY KEY,
  short_id TEXT NOT NULL,
  job_id TEXT NOT NULL,
  env_profile TEXT NOT NULL,
  trigger TEXT NOT NULL,
  actor TEXT,
  status TEXT NOT NULL,
  skip_reason TEXT,
  exit_code INTEGER,
  plan_id TEXT,
  started_at_utc TEXT NOT NULL,
  started_local TEXT NOT NULL,
  started_offset TEXT NOT NULL,
  ended_at_utc TEXT,
  ended_local TEXT,
  ended_offset TEXT,
  log_path TEXT,
  project_version TEXT,
  plan_json TEXT,
  error_summary TEXT,
  pid INTEGER
);
CREATE TABLE IF NOT EXISTS run_steps (
  run_id TEXT NOT NULL,
  step_index INTEGER NOT NULL,
  name TEXT NOT NULL,
  command TEXT NOT NULL,
  output_code_page TEXT,
  detach INTEGER NOT NULL DEFAULT 0,
  exit_code INTEGER,
  duration_ms INTEGER,
  started_local TEXT,
  ended_local TEXT,
  PRIMARY KEY (run_id, step_index)
);
CREATE TABLE IF NOT EXISTS env_audit (
  id TEXT PRIMARY KEY,
  at_utc TEXT NOT NULL,
  at_local TEXT NOT NULL,
  at_offset TEXT NOT NULL,
  actor TEXT,
  job_id TEXT NOT NULL,
  env_profile TEXT NOT NULL,
  key TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS deployments (
  id TEXT PRIMARY KEY,
  deployed_at_utc TEXT NOT NULL,
  deployed_local TEXT NOT NULL,
  deployed_offset TEXT NOT NULL,
  old_version TEXT,
  new_version TEXT,
  source_path TEXT,
  actor TEXT,
  status TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
`

// DB wraps history.db.
type DB struct {
	sql *sql.DB
	path string
}

// Path returns data/history.db.
func Path(dataDir string) string {
	return filepath.Join(dataDir, "history.db")
}

// Open opens or creates history.db in WAL mode with busy_timeout.
func Open(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	p := Path(dataDir)
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)", filepath.ToSlash(p))
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	if _, err := sqlDB.Exec(ddl); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	if _, err := sqlDB.Exec(`INSERT OR IGNORE INTO meta(key,value) VALUES('schema_version',?)`, schemaVersion); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return &DB{sql: sqlDB, path: p}, nil
}

func (d *DB) Close() error {
	if d == nil || d.sql == nil {
		return nil
	}
	return d.sql.Close()
}

func (d *DB) Path() string { return d.path }

// Run is a history row.
type Run struct {
	ID             string  `json:"id"`
	ShortID        string  `json:"short_id"`
	JobID          string  `json:"job_id"`
	EnvProfile     string  `json:"env_profile"`
	Trigger        string  `json:"trigger"`
	Actor          string  `json:"actor,omitempty"`
	Status         string  `json:"status"`
	SkipReason     *string `json:"skip_reason"`
	ExitCode       *int    `json:"exit_code"`
	PlanID         string  `json:"plan_id,omitempty"`
	StartedAtUTC   string  `json:"started_at_utc"`
	StartedLocal   string  `json:"started_local"`
	StartedOffset  string  `json:"started_offset"`
	EndedAtUTC     *string `json:"ended_at_utc"`
	EndedLocal     *string `json:"ended_local"`
	EndedOffset    *string `json:"ended_offset"`
	LogPath        string  `json:"log_path,omitempty"`
	ProjectVersion string  `json:"project_version,omitempty"`
	PlanJSON       string  `json:"plan_json,omitempty"`
	ErrorSummary   *string `json:"error_summary"`
	PID            *int    `json:"pid"`
}

// Step is a run_steps row.
type Step struct {
	RunID          string `json:"run_id"`
	StepIndex      int    `json:"step_index"`
	Name           string `json:"name"`
	Command        string `json:"command"`
	OutputCodePage string `json:"output_code_page,omitempty"`
	Detach         bool   `json:"detach"`
	ExitCode       *int   `json:"exit_code"`
	DurationMS     *int64 `json:"duration_ms"`
	StartedLocal   string `json:"started_local,omitempty"`
	EndedLocal     string `json:"ended_local,omitempty"`
}

// InsertRun inserts a new run row.
func (d *DB) InsertRun(r Run) error {
	_, err := d.sql.Exec(`
INSERT INTO runs(
  id, short_id, job_id, env_profile, trigger, actor, status, skip_reason, exit_code, plan_id,
  started_at_utc, started_local, started_offset, ended_at_utc, ended_local, ended_offset,
  log_path, project_version, plan_json, error_summary, pid
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		r.ID, r.ShortID, r.JobID, r.EnvProfile, r.Trigger, nullStr(r.Actor), r.Status,
		nullStrPtr(r.SkipReason), nullIntPtr(r.ExitCode), nullStr(r.PlanID),
		r.StartedAtUTC, r.StartedLocal, r.StartedOffset,
		nullStrPtr(r.EndedAtUTC), nullStrPtr(r.EndedLocal), nullStrPtr(r.EndedOffset),
		nullStr(r.LogPath), nullStr(r.ProjectVersion), nullStr(r.PlanJSON),
		nullStrPtr(r.ErrorSummary), nullIntPtr(r.PID),
	)
	return err
}

// UpdateRunStatus sets terminal fields for a run.
func (d *DB) UpdateRunStatus(id, status string, exitCode *int, errSummary *string, ended adtime.Instant) error {
	_, err := d.sql.Exec(`
UPDATE runs SET status=?, exit_code=?, error_summary=?,
  ended_at_utc=?, ended_local=?, ended_offset=? WHERE id=?`,
		status, nullIntPtr(exitCode), nullStrPtr(errSummary),
		ended.UTC, ended.Local, ended.Offset, id,
	)
	return err
}

// SetPID updates the live process id for a running row.
func (d *DB) SetPID(id string, pid int) error {
	_, err := d.sql.Exec(`UPDATE runs SET pid=? WHERE id=?`, pid, id)
	return err
}

// StatusOf returns the current status string for a run.
func (d *DB) StatusOf(id string) (string, error) {
	var s string
	err := d.sql.QueryRow(`SELECT status FROM runs WHERE id=?`, id).Scan(&s)
	return s, err
}

// UpsertStep inserts or replaces a step row.
func (d *DB) UpsertStep(s Step) error {
	detach := 0
	if s.Detach {
		detach = 1
	}
	_, err := d.sql.Exec(`
INSERT INTO run_steps(run_id, step_index, name, command, output_code_page, detach, exit_code, duration_ms, started_local, ended_local)
VALUES(?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(run_id, step_index) DO UPDATE SET
  exit_code=excluded.exit_code, duration_ms=excluded.duration_ms,
  started_local=excluded.started_local, ended_local=excluded.ended_local`,
		s.RunID, s.StepIndex, s.Name, s.Command, nullStr(s.OutputCodePage), detach,
		nullIntPtr(s.ExitCode), nullInt64Ptr(s.DurationMS), nullStr(s.StartedLocal), nullStr(s.EndedLocal),
	)
	return err
}

// GetRun returns one run by id.
func (d *DB) GetRun(id string) (Run, error) {
	row := d.sql.QueryRow(`SELECT
  id, short_id, job_id, env_profile, trigger, actor, status, skip_reason, exit_code, plan_id,
  started_at_utc, started_local, started_offset, ended_at_utc, ended_local, ended_offset,
  log_path, project_version, plan_json, error_summary, pid
FROM runs WHERE id=?`, id)
	return scanRun(row)
}

// ListRuns returns recent runs, optionally filtered by job.
func (d *DB) ListRuns(jobID string, limit int) ([]Run, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows *sql.Rows
	var err error
	if jobID == "" {
		rows, err = d.sql.Query(`SELECT
  id, short_id, job_id, env_profile, trigger, actor, status, skip_reason, exit_code, plan_id,
  started_at_utc, started_local, started_offset, ended_at_utc, ended_local, ended_offset,
  log_path, project_version, plan_json, error_summary, pid
FROM runs ORDER BY started_at_utc DESC LIMIT ?`, limit)
	} else {
		rows, err = d.sql.Query(`SELECT
  id, short_id, job_id, env_profile, trigger, actor, status, skip_reason, exit_code, plan_id,
  started_at_utc, started_local, started_offset, ended_at_utc, ended_local, ended_offset,
  log_path, project_version, plan_json, error_summary, pid
FROM runs WHERE job_id=? ORDER BY started_at_utc DESC LIMIT ?`, jobID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Run
	for rows.Next() {
		r, err := scanRunRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListSteps returns steps for a run.
func (d *DB) ListSteps(runID string) ([]Step, error) {
	rows, err := d.sql.Query(`SELECT run_id, step_index, name, command, output_code_page, detach, exit_code, duration_ms, started_local, ended_local
FROM run_steps WHERE run_id=? ORDER BY step_index`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Step
	for rows.Next() {
		var s Step
		var detach int
		var exit sql.NullInt64
		var dur sql.NullInt64
		var ocp, sl, el sql.NullString
		if err := rows.Scan(&s.RunID, &s.StepIndex, &s.Name, &s.Command, &ocp, &detach, &exit, &dur, &sl, &el); err != nil {
			return nil, err
		}
		s.Detach = detach != 0
		s.OutputCodePage = ocp.String
		s.StartedLocal = sl.String
		s.EndedLocal = el.String
		if exit.Valid {
			v := int(exit.Int64)
			s.ExitCode = &v
		}
		if dur.Valid {
			v := dur.Int64
			s.DurationMS = &v
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ActiveRuns returns status=running rows (after optional reconcile).
func (d *DB) ActiveRuns(jobID string) ([]Run, error) {
	q := `SELECT
  id, short_id, job_id, env_profile, trigger, actor, status, skip_reason, exit_code, plan_id,
  started_at_utc, started_local, started_offset, ended_at_utc, ended_local, ended_offset,
  log_path, project_version, plan_json, error_summary, pid
FROM runs WHERE status='running'`
	var rows *sql.Rows
	var err error
	if jobID == "" {
		rows, err = d.sql.Query(q)
	} else {
		rows, err = d.sql.Query(q+` AND job_id=?`, jobID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Run
	for rows.Next() {
		r, err := scanRunRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// InsertEnvAudit records an env set without the value.
func (d *DB) InsertEnvAudit(id, actor, jobID, profile, key string, at adtime.Instant) error {
	_, err := d.sql.Exec(`INSERT INTO env_audit(id, at_utc, at_local, at_offset, actor, job_id, env_profile, key)
VALUES(?,?,?,?,?,?,?,?)`, id, at.UTC, at.Local, at.Offset, actor, jobID, profile, key)
	return err
}

// ReconcileStaleRunning marks running rows whose PID is dead as failed with summary.
func (d *DB) ReconcileStaleRunning() ([]string, error) {
	rows, err := d.ActiveRuns("")
	if err != nil {
		return nil, err
	}
	var fixed []string
	ended := adtime.Now()
	for _, r := range rows {
		stale := false
		if r.PID == nil {
			stale = true
		} else {
			alive, _, aerr := procinfo.Alive(*r.PID)
			if aerr != nil || !alive {
				stale = true
			}
		}
		if !stale {
			continue
		}
		code := 10
		sum := "reconciled: process gone"
		if err := d.UpdateRunStatus(r.ID, "failed", &code, &sum, ended); err != nil {
			return fixed, err
		}
		fixed = append(fixed, r.ID)
	}
	return fixed, nil
}

// Fix is the doctor --fix result.
type Fix struct {
	FixedRuns     []string `json:"fixed_runs"`
	ClearedLocks  []string `json:"cleared_locks"`
}

// ReconcileAll reconciles stale runs and clears stale locks.
func ReconcileAll(dataDir string, d *DB) (Fix, error) {
	fixed, err := d.ReconcileStaleRunning()
	if err != nil {
		return Fix{}, err
	}
	cleared, err := lockfile.ClearStale(dataDir)
	if err != nil {
		return Fix{}, err
	}
	return Fix{FixedRuns: fixed, ClearedLocks: cleared}, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanRun(row scanner) (Run, error) {
	var r Run
	var actor, skip, planID, logPath, pv, planJSON sql.NullString
	var endedUTC, endedLocal, endedOff, errSum sql.NullString
	var exit, pid sql.NullInt64
	err := row.Scan(
		&r.ID, &r.ShortID, &r.JobID, &r.EnvProfile, &r.Trigger, &actor, &r.Status, &skip, &exit, &planID,
		&r.StartedAtUTC, &r.StartedLocal, &r.StartedOffset, &endedUTC, &endedLocal, &endedOff,
		&logPath, &pv, &planJSON, &errSum, &pid,
	)
	if err != nil {
		return r, err
	}
	r.Actor = actor.String
	r.PlanID = planID.String
	r.LogPath = logPath.String
	r.ProjectVersion = pv.String
	r.PlanJSON = planJSON.String
	if skip.Valid {
		r.SkipReason = &skip.String
	}
	if exit.Valid {
		v := int(exit.Int64)
		r.ExitCode = &v
	}
	if endedUTC.Valid {
		r.EndedAtUTC = &endedUTC.String
	}
	if endedLocal.Valid {
		r.EndedLocal = &endedLocal.String
	}
	if endedOff.Valid {
		r.EndedOffset = &endedOff.String
	}
	if errSum.Valid {
		r.ErrorSummary = &errSum.String
	}
	if pid.Valid {
		v := int(pid.Int64)
		r.PID = &v
	}
	return r, nil
}

func scanRunRows(rows *sql.Rows) (Run, error) { return scanRun(rows) }

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
func nullStrPtr(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}
func nullIntPtr(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}
func nullInt64Ptr(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

// InsertDeployment records a promote result.
func (d *DB) InsertDeployment(id, oldVer, newVer, source, actor, status string, at adtime.Instant) error {
	_, err := d.sql.Exec(`INSERT INTO deployments(
  id, deployed_at_utc, deployed_local, deployed_offset, old_version, new_version, source_path, actor, status
) VALUES(?,?,?,?,?,?,?,?,?)`,
		id, at.UTC, at.Local, at.Offset, oldVer, newVer, source, actor, status)
	return err
}

// NewRunID returns a unique run id (UUID, no dashes) and an 8-char short id
// taken from the random hex (plan §4.8 / CONTRACT: ULID or UUID). Not PID-based.
func NewRunID() (id, short string) {
	u := uuid.New()
	id = strings.ReplaceAll(u.String(), "-", "")
	short = id[:8]
	return id, short
}
