package envfile

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PrakashNiraula/autodeployer/internal/machine"
)

func Dir(dataDir, jobID string) string {
	return filepath.Join(dataDir, "env", jobID)
}

func Path(dataDir, jobID, profile string) string {
	return filepath.Join(Dir(dataDir, jobID), ".env."+profile)
}

func ExamplePath(jobDir, profile string) string {
	return filepath.Join(jobDir, "env", ".env."+profile+".example")
}

func ParseFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		v = strings.Trim(v, `"'`)
		out[k] = v
	}
	return out, sc.Err()
}

type LoadResult struct {
	Values  map[string]string
	Missing []string
	Empty   []string
	Source  string
	OK      bool
}

func LoadForRun(dataDir, jobDir, jobID, profile string, required []string, cfg machine.Config, allowExampleFlag bool) (LoadResult, error) {
	res := LoadResult{Values: map[string]string{}}
	if !cfg.Allows(profile) {
		return res, fmt.Errorf("environment %q is not allowed on this machine", profile)
	}
	p := Path(dataDir, jobID, profile)
	b, err := os.ReadFile(p)
	useExample := false
	if err != nil {
		if os.IsNotExist(err) {
			if allowExampleFlag && profile == "dev" && cfg.AllowExampleEnv {
				ep := ExamplePath(jobDir, profile)
				b, err = os.ReadFile(ep)
				if err != nil {
					return res, fmt.Errorf("env file missing: %s (and no usable .example)", p)
				}
				useExample = true
				res.Source = ep
			} else {
				return res, fmt.Errorf("env file missing: %s", p)
			}
		} else {
			return res, err
		}
	} else {
		res.Source = p
		_ = b
	}
	vals, err := ParseFile(res.Source)
	if err != nil {
		return res, err
	}
	if useExample && !(allowExampleFlag && profile == "dev" && cfg.AllowExampleEnv) {
		return res, fmt.Errorf("refusing to use .example env for runs")
	}
	res.Values = vals
	for _, k := range required {
		v, ok := vals[k]
		if !ok {
			res.Missing = append(res.Missing, k)
			continue
		}
		if strings.TrimSpace(v) == "" {
			res.Empty = append(res.Empty, k)
		}
	}
	res.OK = len(res.Missing) == 0 && len(res.Empty) == 0
	return res, nil
}

func SetKey(dataDir, jobID, profile, key, value string) error {
	dir := Dir(dataDir, jobID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	p := Path(dataDir, jobID, profile)
	vals := map[string]string{}
	if _, err := os.Stat(p); err == nil {
		var e error
		vals, e = ParseFile(p)
		if e != nil {
			return e
		}
	}
	vals[key] = value
	return writeDotEnv(p, vals)
}

func writeDotEnv(path string, vals map[string]string) error {
	var b strings.Builder
	keys := make([]string, 0, len(vals))
	for k := range vals {
		keys = append(keys, k)
	}
	// stable-ish order
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%s\n", k, vals[k])
	}
	return os.WriteFile(path, []byte(b.String()), 0o600)
}
