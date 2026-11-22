package plan

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/PrakashNiraula/autodeployer/internal/envfile"
	"github.com/PrakashNiraula/autodeployer/internal/job"
	"github.com/PrakashNiraula/autodeployer/internal/machine"
	"github.com/PrakashNiraula/autodeployer/internal/mask"
)

type Result struct {
	PlanID         string            `json:"plan_id"`
	JobID          string            `json:"job_id"`
	JobName        string            `json:"job_name"`
	EnvProfile     string            `json:"env_profile"`
	MachineName    string            `json:"machine_name"`
	ProjectVersion string            `json:"project_version"`
	Workdir        string            `json:"workdir"`
	TimeoutMinutes int               `json:"timeout_minutes"`
	Encoding       map[string]string `json:"encoding"`
	Steps          []StepOut         `json:"steps"`
	EnvValidation  EnvValidation     `json:"env_validation"`
	ActiveRuns     []ActiveRun       `json:"active_runs"`
	Warnings       []string          `json:"warnings"`
	CanExecute     bool              `json:"can_execute"`
}

type StepOut struct {
	Index          int    `json:"index"`
	Name           string `json:"name"`
	Command        string `json:"command"`
	OutputCodePage string `json:"output_code_page,omitempty"`
	Detach         bool   `json:"detach"`
}

type EnvValidation struct {
	OK             bool              `json:"ok"`
	Missing        []string          `json:"missing"`
	Empty          []string          `json:"empty"`
	PresentMasked  map[string]string `json:"present_masked"`
}

type ActiveRun struct {
	RunID  string `json:"run_id"`
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

func Build(root string, m *job.Manifest, profile string, cfg machine.Config, allowExample bool, active []ActiveRun) (Result, error) {
	if !cfg.Allows(profile) {
		return Result{}, fmt.Errorf("ENV_NOT_ALLOWED")
	}
	dataDir := filepath.Join(root, "data")
	loaded, err := envfile.LoadForRun(dataDir, m.Dir, m.ID, profile, m.RequiredEnv, cfg, allowExample)
	if err != nil {
		if strings.Contains(err.Error(), "not allowed") {
			return Result{}, fmt.Errorf("ENV_NOT_ALLOWED")
		}
		// missing file → treat as validation fail with missing all keys
		if strings.Contains(err.Error(), "missing") {
			loaded.Missing = append([]string{}, m.RequiredEnv...)
			loaded.Values = map[string]string{}
			loaded.OK = false
		} else {
			return Result{}, err
		}
	}
	vr := m.Validate()
	cp := m.Encoding.OutputCodePage
	if cp == "" || cp == "auto" {
		cp = cfg.DefaultOutputCodePage
	}
	steps := make([]StepOut, 0, len(m.Steps))
	for i, s := range m.Steps {
		scp := s.OutputCodePage
		if scp == "" {
			scp = cp
		}
		steps = append(steps, StepOut{Index: i, Name: s.Name, Command: s.Command, OutputCodePage: scp, Detach: s.Detach})
	}
	masked := mask.MaskMap(loaded.Values, m.Secrets, cfg.SecretMinMaskLength)
	// only include required keys in present_masked for clarity
	present := map[string]string{}
	for _, k := range m.RequiredEnv {
		if v, ok := masked[k]; ok {
			present[k] = v
		}
	}
	pv := readProjectVersion(root)
	can := loaded.OK && vr.OK
	res := Result{
		JobID:          m.ID,
		JobName:        m.Name,
		EnvProfile:     profile,
		MachineName:    cfg.MachineName,
		ProjectVersion: pv,
		Workdir:        m.WorkdirAbs(),
		TimeoutMinutes: m.TimeoutMinutes,
		Encoding:       map[string]string{"output_code_page": cp},
		Steps:          steps,
		EnvValidation: EnvValidation{
			OK:            loaded.OK,
			Missing:       loaded.Missing,
			Empty:         loaded.Empty,
			PresentMasked: present,
		},
		ActiveRuns: active,
		Warnings:   vr.Warnings,
		CanExecute: can,
	}
	res.PlanID = Hash(m, profile, loaded.Values, pv)
	return res, nil
}

func Hash(m *job.Manifest, profile string, values map[string]string, projectVersion string) string {
	h := sha256.New()
	raw, _ := os.ReadFile(filepath.Join(m.Dir, "job.yaml"))
	h.Write(raw)
	h.Write([]byte{0})
	h.Write([]byte(projectVersion))
	h.Write([]byte{0})
	h.Write([]byte(profile))
	h.Write([]byte{0})
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte{0})
		h.Write([]byte(values[k]))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func readProjectVersion(root string) string {
	b, err := os.ReadFile(filepath.Join(root, "data", "project-version.json"))
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
