package machine

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is data/machine.yaml (plan 4.16).
type Config struct {
	MachineName                  string   `yaml:"machine_name" json:"machine_name"`
	AllowedEnvs                  []string `yaml:"allowed_envs" json:"allowed_envs"`
	DefaultEnv                   string   `yaml:"default_env" json:"default_env"`
	DefaultOutputCodePage        string   `yaml:"default_output_code_page" json:"default_output_code_page"`
	EnvFileEncoding              string   `yaml:"env_file_encoding" json:"env_file_encoding"`
	LogRetentionDays             int      `yaml:"log_retention_days" json:"log_retention_days"`
	AllowExampleEnv              bool     `yaml:"allow_example_env" json:"allow_example_env"`
	PromoteWaitSecondsManual     int      `yaml:"promote_wait_seconds_manual" json:"promote_wait_seconds_manual"`
	PromoteWaitSecondsScheduled  int      `yaml:"promote_wait_seconds_scheduled" json:"promote_wait_seconds_scheduled"`
	SecretMinMaskLength          int      `yaml:"secret_min_mask_length" json:"secret_min_mask_length"`
	Notify                       Notify   `yaml:"notify" json:"notify"`
}

type Notify struct {
	OnFailure   bool   `yaml:"on_failure" json:"on_failure"`
	OnSkipped   bool   `yaml:"on_skipped" json:"on_skipped"`
	WebhookURL  string `yaml:"webhook_url" json:"webhook_url"`
}

func Defaults() Config {
	return Config{
		MachineName:                 "local",
		AllowedEnvs:                 []string{"dev"},
		DefaultEnv:                  "dev",
		DefaultOutputCodePage:       "utf-8",
		EnvFileEncoding:             "utf-8",
		LogRetentionDays:            30,
		AllowExampleEnv:             false,
		PromoteWaitSecondsManual:    30,
		PromoteWaitSecondsScheduled: 300,
		SecretMinMaskLength:         4,
	}
}

func Path(dataDir string) string {
	return filepath.Join(dataDir, "machine.yaml")
}

// Load reads machine.yaml or returns defaults if missing.
func Load(dataDir string) (Config, error) {
	cfg := Defaults()
	p := Path(dataDir)
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("machine.yaml: %w", err)
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("machine.yaml: invalid YAML: %w", err)
	}
	applyDefaults(&cfg)
	if err := validate(cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func applyDefaults(c *Config) {
	d := Defaults()
	if c.MachineName == "" {
		c.MachineName = d.MachineName
	}
	if len(c.AllowedEnvs) == 0 {
		c.AllowedEnvs = d.AllowedEnvs
	}
	if c.DefaultEnv == "" {
		c.DefaultEnv = c.AllowedEnvs[0]
	}
	if c.DefaultOutputCodePage == "" {
		c.DefaultOutputCodePage = d.DefaultOutputCodePage
	}
	if c.EnvFileEncoding == "" {
		c.EnvFileEncoding = d.EnvFileEncoding
	}
	if c.LogRetentionDays <= 0 {
		c.LogRetentionDays = d.LogRetentionDays
	}
	if c.PromoteWaitSecondsManual <= 0 {
		c.PromoteWaitSecondsManual = d.PromoteWaitSecondsManual
	}
	if c.PromoteWaitSecondsScheduled <= 0 {
		c.PromoteWaitSecondsScheduled = d.PromoteWaitSecondsScheduled
	}
	if c.SecretMinMaskLength <= 0 {
		c.SecretMinMaskLength = d.SecretMinMaskLength
	}
}

func validate(c Config) error {
	if c.DefaultEnv == "" {
		return fmt.Errorf("machine.yaml: default_env is empty")
	}
	ok := false
	for _, e := range c.AllowedEnvs {
		if e == c.DefaultEnv {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("machine.yaml: default_env %q is not in allowed_envs", c.DefaultEnv)
	}
	return nil
}

func (c Config) Allows(env string) bool {
	for _, e := range c.AllowedEnvs {
		if e == env {
			return true
		}
	}
	return false
}
