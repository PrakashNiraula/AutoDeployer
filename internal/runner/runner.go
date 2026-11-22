// Package runner executes planned jobs (fail-fast, encoding, locks, history).
package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/PrakashNiraula/autodeployer/internal/adtime"
	"github.com/PrakashNiraula/autodeployer/internal/disklocal"
	"github.com/PrakashNiraula/autodeployer/internal/encode"
	"github.com/PrakashNiraula/autodeployer/internal/envfile"
	"github.com/PrakashNiraula/autodeployer/internal/exitcode"
	"github.com/PrakashNiraula/autodeployer/internal/job"
	"github.com/PrakashNiraula/autodeployer/internal/killtree"
	"github.com/PrakashNiraula/autodeployer/internal/lockfile"
	"github.com/PrakashNiraula/autodeployer/internal/machine"
	"github.com/PrakashNiraula/autodeployer/internal/mask"
	"github.com/PrakashNiraula/autodeployer/internal/plan"
	"github.com/PrakashNiraula/autodeployer/internal/store"
)

// Options for a single run invocation.
type Options struct {
	Root         string
	Job          *job.Manifest
	EnvProfile   string
	Trigger      string
	PlanID       string
	AllowExample bool
	Yes          bool
	JSONMeta     string
	Events       string
	Actor        string
	Quiet        bool
}

// Result mirrors RunResult contract fields needed by CLI.
type Result struct {
	RunID          string       `json:"run_id"`
	ShortID        string       `json:"short_id"`
	JobID          string       `json:"job_id"`
	EnvProfile     string       `json:"env_profile"`
	Trigger        string       `json:"trigger"`
	Actor          string       `json:"actor,omitempty"`
	Status         string       `json:"status"`
	SkipReason     *string      `json:"skip_reason"`
	ExitCode       *int         `json:"exit_code"`
	PlanID         string       `json:"plan_id"`
	ProjectVersion string       `json:"project_version"`
	LogPath        string       `json:"log_path"`
	Started        string       `json:"started"`
	Ended          *string      `json:"ended"`
	ErrorSummary   *string      `json:"error_summary"`
	Steps          []StepResult `json:"steps"`
	CLIExit        int          `json:"-"`
}

type StepResult struct {
	Index          int    `json:"index"`
	Name           string `json:"name"`
	Command        string `json:"command"`
	OutputCodePage string `json:"output_code_page"`
	Detach         bool   `json:"detach"`
	ExitCode       *int   `json:"exit_code"`
	DurationMS     *int64 `json:"duration_ms"`
}

// Execute runs a job end-to-end.
func Execute(opts Options) (Result, error) {
	dataDir := filepath.Join(opts.Root, "data")
	cfg, err := machine.Load(dataDir)
	if err != nil {
		return Result{CLIExit: exitcode.Schema}, err
	}
	if err := disklocal.EnsureLocal(dataDir); err != nil {
		return Result{CLIExit: exitcode.Internal}, err
	}

	db, err := store.Open(dataDir)
	if err != nil {
		return Result{CLIExit: exitcode.Internal}, err
	}
	defer db.Close()
	_, _ = db.ReconcileStaleRunning()

	active, _ := db.ActiveRuns(opts.Job.ID)
	var activePlan []plan.ActiveRun
	for _, r := range active {
		activePlan = append(activePlan, plan.ActiveRun{RunID: r.ID, JobID: r.JobID, Status: r.Status})
	}
	pl, err := plan.Build(opts.Root, opts.Job, opts.EnvProfile, cfg, opts.AllowExample, activePlan)
	if err != nil {
		if err.Error() == "ENV_NOT_ALLOWED" {
			return Result{CLIExit: exitcode.EnvNotAllowed}, err
		}
		return Result{CLIExit: exitcode.Internal}, err
	}
	if !pl.EnvValidation.OK {
		return Result{CLIExit: exitcode.EnvValidation}, fmt.Errorf("env validation failed")
	}
	if opts.PlanID != "" && opts.PlanID != pl.PlanID {
		return Result{CLIExit: exitcode.PlanMismatch}, fmt.Errorf("plan_id mismatch")
	}

	loaded, err := envfile.LoadForRun(dataDir, opts.Job.Dir, opts.Job.ID, opts.EnvProfile, opts.Job.RequiredEnv, cfg, opts.AllowExample)
	if err != nil || !loaded.OK {
		return Result{CLIExit: exitcode.EnvValidation}, fmt.Errorf("env validation failed")
	}

	wait := time.Duration(cfg.PromoteWaitSecondsManual) * time.Second
	if opts.Trigger == "scheduled" {
		wait = time.Duration(cfg.PromoteWaitSecondsScheduled) * time.Second
	}
	if err := lockfile.WaitPromote(dataDir, wait); err != nil {
		return recordSkipped(db, opts, pl, "promote_lock", exitcode.PromoteLockHeld)
	}

	runID, shortID := store.NewRunID()
	actor := opts.Actor
	if actor == "" {
		actor = opts.Trigger
	}

	if !opts.Job.AllowConcurrent {
		if _, err := lockfile.Acquire(dataDir, opts.Job.ID, runID, actor); err != nil {
			if errors.Is(err, lockfile.ErrHeld) {
				return recordSkipped(db, opts, pl, "job_lock", exitcode.JobLocked)
			}
			return Result{CLIExit: exitcode.Internal}, err
		}
		defer func() { _ = lockfile.Release(dataDir, opts.Job.ID, runID) }()
	}

	started := adtime.Now()
	outDir := filepath.Join(dataDir, "output", opts.Job.ID, runID)
	_ = os.MkdirAll(outDir, 0o755)
	_ = os.MkdirAll(filepath.Join(dataDir, "logs", opts.Job.ID), 0o755)
	logName := fmt.Sprintf("%s_%s_%s.log", started.LogFileStamp(), shortID, opts.Trigger)
	logPath := filepath.Join(dataDir, "logs", opts.Job.ID, logName)

	planJSON, _ := json.Marshal(pl)
	pid := os.Getpid()
	row := store.Run{
		ID: runID, ShortID: shortID, JobID: opts.Job.ID, EnvProfile: opts.EnvProfile,
		Trigger: opts.Trigger, Actor: actor, Status: "running", PlanID: pl.PlanID,
		StartedAtUTC: started.UTC, StartedLocal: started.Local, StartedOffset: started.Offset,
		LogPath: logPath, ProjectVersion: pl.ProjectVersion, PlanJSON: string(planJSON), PID: &pid,
	}
	if err := db.InsertRun(row); err != nil {
		return Result{CLIExit: exitcode.Internal}, err
	}

	logFile, err := os.Create(logPath)
	if err != nil {
		return Result{CLIExit: exitcode.Internal}, err
	}
	defer logFile.Close()

	writeHeader(logFile, row, pl, started.Canonical)
	emitEvent(opts.Events, map[string]any{"type": "run_start", "run_id": runID, "job_id": opts.Job.ID})

	envVals := cloneMap(loaded.Values)
	envVals["AD_RUN_ID"] = runID

	res := Result{
		RunID: runID, ShortID: shortID, JobID: opts.Job.ID, EnvProfile: opts.EnvProfile,
		Trigger: opts.Trigger, Actor: actor, Status: "running", PlanID: pl.PlanID,
		ProjectVersion: pl.ProjectVersion, LogPath: logPath, Started: started.Canonical,
		Steps: []StepResult{},
	}

	timeout := time.Duration(opts.Job.TimeoutMinutes) * time.Minute
	if timeout <= 0 {
		timeout = 60 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var runPID int
	status := "succeeded"
	cliExit := exitcode.OK
	var errSummary *string
	var lastExit *int

	secretValues := secretValueList(envVals, opts.Job.Secrets)
	minMask := cfg.SecretMinMaskLength

	for i, step := range pl.Steps {
		stepStart := adtime.Now()
		emitEvent(opts.Events, map[string]any{"type": "step_start", "run_id": runID, "index": i, "name": step.Name})
		_ = db.UpsertStep(store.Step{
			RunID: runID, StepIndex: i, Name: step.Name, Command: step.Command,
			OutputCodePage: step.OutputCodePage, Detach: step.Detach, StartedLocal: stepStart.Local,
		})

		code, dur, stepPID, _ := runStep(ctx, opts, runID, outDir, step, envVals, logFile, secretValues, minMask, !opts.Quiet)
		if stepPID > 0 {
			runPID = stepPID
			_ = db.SetPID(runID, stepPID)
		}
		ec := code
		dms := dur.Milliseconds()
		res.Steps = append(res.Steps, StepResult{
			Index: i, Name: step.Name, Command: step.Command, OutputCodePage: step.OutputCodePage,
			Detach: step.Detach, ExitCode: &ec, DurationMS: &dms,
		})
		endedStep := adtime.Now()
		_ = db.UpsertStep(store.Step{
			RunID: runID, StepIndex: i, Name: step.Name, Command: step.Command,
			OutputCodePage: step.OutputCodePage, Detach: step.Detach,
			ExitCode: &ec, DurationMS: &dms, StartedLocal: stepStart.Local, EndedLocal: endedStep.Local,
		})
		emitEvent(opts.Events, map[string]any{"type": "step_end", "run_id": runID, "index": i, "exit_code": code})

		if ctx.Err() == context.DeadlineExceeded {
			status = "timeout"
			cliExit = exitcode.Timeout
			if runPID > 0 {
				_ = killtree.KillProcessGroup(runPID)
			}
			s := "timeout"
			errSummary = &s
			lastExit = &cliExit
			break
		}
		if code != 0 {
			status = "failed"
			cliExit = exitcode.JobFailed
			lastExit = &code
			s := fmt.Sprintf("step %d %q exited %d", i, step.Name, code)
			errSummary = &s
			break
		}
		lastExit = &code
	}

	ended := adtime.Now()
	endedCanon := ended.Canonical
	if cur, err := db.StatusOf(runID); err == nil && cur == "cancelled" {
		status = "cancelled"
		cliExit = exitcode.Cancelled
		c := exitcode.Cancelled
		lastExit = &c
		s := "cancelled"
		errSummary = &s
	}
	res.Status = status
	res.Ended = &endedCanon
	res.ExitCode = lastExit
	res.ErrorSummary = errSummary
	res.CLIExit = cliExit
	_ = db.UpdateRunStatus(runID, status, lastExit, errSummary, ended)
	emitEvent(opts.Events, map[string]any{"type": "run_end", "run_id": runID, "status": status})
	writeMeta(opts.JSONMeta, res)
	return res, nil
}

func recordSkipped(db *store.DB, opts Options, pl plan.Result, reason string, cli int) (Result, error) {
	runID, shortID := store.NewRunID()
	started := adtime.Now()
	rs := reason
	ec := cli
	row := store.Run{
		ID: runID, ShortID: shortID, JobID: opts.Job.ID, EnvProfile: opts.EnvProfile,
		Trigger: opts.Trigger, Actor: opts.Actor, Status: "skipped", SkipReason: &rs, ExitCode: &ec,
		PlanID: pl.PlanID, StartedAtUTC: started.UTC, StartedLocal: started.Local, StartedOffset: started.Offset,
		EndedAtUTC: &started.UTC, EndedLocal: &started.Local, EndedOffset: &started.Offset,
		ProjectVersion: pl.ProjectVersion,
	}
	_ = db.InsertRun(row)
	ended := started.Canonical
	res := Result{
		RunID: runID, ShortID: shortID, JobID: opts.Job.ID, EnvProfile: opts.EnvProfile,
		Trigger: opts.Trigger, Status: "skipped", SkipReason: &rs, ExitCode: &ec,
		PlanID: pl.PlanID, ProjectVersion: pl.ProjectVersion, Started: started.Canonical, Ended: &ended,
		CLIExit: cli, Steps: []StepResult{},
	}
	writeMeta(opts.JSONMeta, res)
	return res, fmt.Errorf("skipped: %s", reason)
}

func runStep(ctx context.Context, opts Options, runID, outDir string, step plan.StepOut, envVals map[string]string, logFile *os.File, secretValues []string, minMask int, tee bool) (exitCode int, dur time.Duration, pid int, err error) {
	argv, err := ResolveCommand(opts.Job.WorkdirAbs(), step.Command)
	if err != nil {
		fmt.Fprintf(logFile, "[autodeployer] resolve: %v\n", err)
		return 1, 0, 0, err
	}
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = opts.Job.WorkdirAbs()
	cmd.Env = buildEnv(opts, runID, outDir, envVals)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 1, 0, 0, err
	}
	cmd.Stderr = cmd.Stdout

	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}

	start := time.Now()
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(logFile, "[autodeployer] start: %v\n", err)
		return 1, time.Since(start), 0, err
	}
	pid = cmd.Process.Pid

	dec := encode.NewLineDecoder(step.OutputCodePage)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, rerr := stdout.Read(buf)
			if n > 0 {
				decoded := dec.DecodeChunk(buf[:n])
				text := mask.Apply(string(decoded), secretValues, minMask)
				_, _ = io.WriteString(logFile, text)
				if tee {
					_, _ = io.WriteString(os.Stdout, text)
				}
			}
			if rerr != nil {
				break
			}
		}
		if tail := dec.Flush(); len(tail) > 0 {
			text := mask.Apply(string(tail), secretValues, minMask)
			_, _ = io.WriteString(logFile, text)
			if tee {
				_, _ = io.WriteString(os.Stdout, text)
			}
		}
	}()

	waitErr := cmd.Wait()
	wg.Wait()
	dur = time.Since(start)

	if ctx.Err() == context.DeadlineExceeded {
		_ = killtree.KillProcessGroup(pid)
		return exitcode.Timeout, dur, pid, ctx.Err()
	}
	if waitErr == nil {
		return 0, dur, pid, nil
	}
	var ee *exec.ExitError
	if errors.As(waitErr, &ee) {
		return ee.ExitCode(), dur, pid, nil
	}
	return 1, dur, pid, waitErr
}

func buildEnv(opts Options, runID, outDir string, vals map[string]string) []string {
	extra := map[string]string{
		"AD_ROOT":           opts.Root,
		"AD_DATA_DIR":       filepath.Join(opts.Root, "data"),
		"AD_JOB_OUTPUT_DIR": outDir,
		"AD_RUN_ID":         runID,
		"AD_ENV":            opts.EnvProfile,
		"AD_JOB_ID":         opts.Job.ID,
	}
	for k, v := range vals {
		extra[k] = v
	}
	return mergeEnv(os.Environ(), extra)
}

func mergeEnv(base []string, extra map[string]string) []string {
	m := map[string]string{}
	for _, e := range base {
		if k, v, ok := strings.Cut(e, "="); ok {
			m[k] = v
		}
	}
	for k, v := range extra {
		m[k] = v
	}
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	return out
}

func writeHeader(w io.Writer, row store.Run, pl plan.Result, started string) {
	fmt.Fprintf(w, "=== AutoDeployer run ===\n")
	fmt.Fprintf(w, "run_id: %s\nshort_id: %s\njob: %s\nenv: %s\ntrigger: %s\nmachine: %s\nproject_version: %s\nplan_id: %s\nstarted: %s\n",
		row.ID, row.ShortID, row.JobID, row.EnvProfile, row.Trigger, pl.MachineName, row.ProjectVersion, row.PlanID, started)
	fmt.Fprintf(w, "plan_json: %s\n", row.PlanJSON)
	fmt.Fprintf(w, "=== stdout/stderr follow (UTF-8) ===\n")
}

func writeMeta(path string, res Result) {
	if path == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	b, _ := json.MarshalIndent(res, "", "  ")
	_ = os.WriteFile(path, b, 0o644)
}

func emitEvent(path string, ev map[string]any) {
	if path == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	b, _ := json.Marshal(ev)
	_, _ = f.Write(append(b, '\n'))
	_ = f.Sync()
}

func cloneMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func secretValueList(vals map[string]string, declared []string) []string {
	var out []string
	for k, v := range vals {
		if mask.IsSecretKey(k, declared) {
			out = append(out, v)
		}
	}
	return out
}

// Cancel tree-kills a running job and finalizes history.
func Cancel(root, runID string) (Result, int, error) {
	dataDir := filepath.Join(root, "data")
	db, err := store.Open(dataDir)
	if err != nil {
		return Result{}, exitcode.Internal, err
	}
	defer db.Close()
	run, err := db.GetRun(runID)
	if err != nil {
		return Result{}, exitcode.CancelNotFound, fmt.Errorf("run not found")
	}
	if run.Status != "running" {
		return Result{}, exitcode.CancelNotFound, fmt.Errorf("run not running")
	}
	pid := 0
	if run.PID != nil {
		pid = *run.PID
	}
	if held, info, _ := lockfile.IsHeld(dataDir, run.JobID); held && info.RunID == runID {
		pid = info.PID
	}
	if pid > 0 {
		_ = killtree.KillProcessGroup(pid)
	}
	ended := adtime.Now()
	code := exitcode.Cancelled
	sum := "cancelled"
	_ = db.UpdateRunStatus(runID, "cancelled", &code, &sum, ended)
	_ = lockfile.Release(dataDir, run.JobID, runID)
	endedC := ended.Canonical
	res := Result{
		RunID: run.ID, ShortID: run.ShortID, JobID: run.JobID, EnvProfile: run.EnvProfile,
		Trigger: run.Trigger, Status: "cancelled", ExitCode: &code, PlanID: run.PlanID,
		LogPath: run.LogPath, Started: run.StartedLocal + run.StartedOffset, Ended: &endedC,
		ErrorSummary: &sum, CLIExit: exitcode.OK, Steps: []StepResult{},
	}
	return res, exitcode.OK, nil
}
