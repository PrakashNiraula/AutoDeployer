package job

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

type Manifest struct {
	SchemaVersion   int      `yaml:"schema_version" json:"schema_version"`
	ID              string   `yaml:"id" json:"id"`
	Name            string   `yaml:"name" json:"name"`
	Description     string   `yaml:"description" json:"description"`
	Workdir         string   `yaml:"workdir" json:"workdir"`
	Shell           string   `yaml:"shell" json:"shell"`
	Encoding        Encoding `yaml:"encoding" json:"encoding"`
	Steps           []Step   `yaml:"steps" json:"steps"`
	RequiredEnv     []string `yaml:"required_env" json:"required_env"`
	Secrets         []string `yaml:"secrets" json:"secrets"`
	TimeoutMinutes  int      `yaml:"timeout_minutes" json:"timeout_minutes"`
	AllowConcurrent bool     `yaml:"allow_concurrent" json:"allow_concurrent"`
	LogPaths        []string `yaml:"log_paths" json:"log_paths"`

	Dir string `yaml:"-" json:"-"` // absolute job folder
}

type Encoding struct {
	OutputCodePage string `yaml:"output_code_page" json:"output_code_page"`
}

type Step struct {
	Name           string `yaml:"name" json:"name"`
	Command        string `yaml:"command" json:"command"`
	OutputCodePage string `yaml:"output_code_page" json:"output_code_page"`
	Detach         bool   `yaml:"detach" json:"detach"`
}

type Summary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
}

type ValidationResult struct {
	OK       bool     `json:"ok"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

func Load(jobDir string) (*Manifest, error) {
	p := filepath.Join(jobDir, "job.yaml")
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("job.yaml: cannot read %s: %w", p, err)
	}
	var m Manifest
	if err := yaml.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("job.yaml: invalid YAML in %s: %w", p, err)
	}
	abs, err := filepath.Abs(jobDir)
	if err != nil {
		return nil, err
	}
	m.Dir = abs
	return &m, nil
}

func Discover(jobsRoot string) ([]Summary, error) {
	entries, err := os.ReadDir(jobsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("jobs folder: %w", err)
	}
	var out []Summary
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(jobsRoot, e.Name())
		m, err := Load(dir)
		if err != nil {
			continue // skip unreadable; validate surfaces errors
		}
		out = append(out, Summary{
			ID:          m.ID,
			Name:        m.Name,
			Description: strings.TrimSpace(m.Description),
			Path:        dir,
		})
	}
	return out, nil
}

func Find(jobsRoot, id string) (*Manifest, error) {
	dir := filepath.Join(jobsRoot, id)
	m, err := Load(dir)
	if err != nil {
		return nil, fmt.Errorf("job %q: %w", id, err)
	}
	return m, nil
}

func (m *Manifest) Validate() ValidationResult {
	var errs, warns []string
	folder := filepath.Base(m.Dir)
	if m.SchemaVersion != 1 {
		errs = append(errs, fmt.Sprintf("job.yaml: schema_version: want 1, got %d", m.SchemaVersion))
	}
	if m.ID == "" {
		errs = append(errs, "job.yaml: id: missing")
	} else if m.ID != folder {
		errs = append(errs, fmt.Sprintf("job.yaml: id: %q does not match folder name %q", m.ID, folder))
	}
	if m.Name == "" {
		errs = append(errs, "job.yaml: name: missing")
	}
	if m.Workdir == "" {
		errs = append(errs, "job.yaml: workdir: missing")
	} else if err := pathInsideJob(m.Dir, m.Workdir); err != nil {
		errs = append(errs, fmt.Sprintf("job.yaml: workdir: %s", err))
	}
	if len(m.Steps) == 0 {
		errs = append(errs, "job.yaml: steps: at least one step required")
	}
	secretSet := map[string]bool{}
	reqSet := map[string]bool{}
	for _, r := range m.RequiredEnv {
		reqSet[r] = true
	}
	for _, s := range m.Secrets {
		secretSet[s] = true
		if !reqSet[s] {
			errs = append(errs, fmt.Sprintf("job.yaml: secrets: %q is not in required_env", s))
		}
	}
	workdirAbs := filepath.Join(m.Dir, m.Workdir)
	for i, st := range m.Steps {
		prefix := fmt.Sprintf("job.yaml: steps[%d]", i)
		if st.Name == "" {
			errs = append(errs, prefix+".name: missing")
		}
		if st.Command == "" {
			errs = append(errs, prefix+".command: missing")
			continue
		}
		if err := pathInsideJob(m.Dir, filepath.Join(m.Workdir, st.Command)); err != nil {
			// also try command as relative to job root
			if err2 := pathInsideJob(m.Dir, st.Command); err2 != nil {
				errs = append(errs, fmt.Sprintf("%s.command: %s", prefix, err))
				continue
			}
		}
		cmdPath := resolveCommandPath(m.Dir, workdirAbs, st.Command)
		if cmdPath == "" || !fileExists(cmdPath) {
			errs = append(errs, fmt.Sprintf("%s.command: file not found: %s", prefix, st.Command))
		} else if strings.HasSuffix(strings.ToLower(st.Command), ".bat") {
			if w := lintBatCall(cmdPath); w != "" {
				warns = append(warns, fmt.Sprintf("%s: %s", st.Command, w))
			}
		}
	}
	return ValidationResult{OK: len(errs) == 0, Errors: errs, Warnings: warns}
}

func resolveCommandPath(jobDir, workdirAbs, command string) string {
	candidates := []string{
		filepath.Join(workdirAbs, command),
		filepath.Join(jobDir, command),
	}
	if runtime.GOOS != "windows" {
		if !strings.Contains(command, ".") {
			candidates = append(candidates,
				filepath.Join(workdirAbs, command+".sh"),
				filepath.Join(jobDir, command+".sh"),
			)
		}
	}
	for _, c := range candidates {
		if fileExists(c) {
			return c
		}
	}
	return filepath.Join(workdirAbs, command)
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func pathInsideJob(jobDir, relOrAbs string) error {
	if strings.Contains(relOrAbs, "..") {
		return fmt.Errorf("path escapes job folder: %q", relOrAbs)
	}
	var target string
	if filepath.IsAbs(relOrAbs) {
		target = filepath.Clean(relOrAbs)
	} else {
		target = filepath.Clean(filepath.Join(jobDir, relOrAbs))
	}
	jobAbs, err := filepath.Abs(jobDir)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(jobAbs, target)
	if err != nil {
		return fmt.Errorf("cannot resolve path %q", relOrAbs)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("path escapes job folder: %q", relOrAbs)
	}
	return nil
}

func lintBatCall(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(b), "\n")
	for _, line := range lines {
		t := strings.TrimSpace(line)
		low := strings.ToLower(t)
		if strings.HasPrefix(low, "rem") || strings.HasPrefix(low, "::") {
			continue
		}
		if strings.HasSuffix(low, ".bat") && !strings.HasPrefix(low, "call ") {
			if !strings.Contains(low, " ") || strings.Count(low, " ") >= 0 {
				// bare nested bat invocation heuristic
				if !strings.Contains(low, "=") && strings.Contains(low, ".bat") && !strings.HasPrefix(low, "call") {
					parts := strings.Fields(low)
					if len(parts) >= 1 && strings.HasSuffix(parts[0], ".bat") {
						return "nested .bat without call may hide exit codes; prefer call script.bat"
					}
				}
			}
		}
	}
	return ""
}

// WorkdirAbs returns absolute workdir for the job.
func (m *Manifest) WorkdirAbs() string {
	return filepath.Join(m.Dir, m.Workdir)
}
