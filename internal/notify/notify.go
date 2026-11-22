// Package notify sends optional webhooks for terminal run states.
package notify

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/PrakashNiraula/autodeployer/internal/machine"
)

// ShouldFire reports whether status warrants a notification.
func ShouldFire(cfg machine.Config, status string) bool {
	if cfg.Notify.WebhookURL == "" {
		return false
	}
	switch status {
	case "failed", "timeout", "cancelled":
		return cfg.Notify.OnFailure
	case "skipped":
		return cfg.Notify.OnSkipped
	default:
		return false
	}
}

// Fire posts a JSON payload to the configured webhook (best-effort).
func Fire(cfg machine.Config, payload map[string]any) error {
	url := cfg.Notify.WebhookURL
	if url == "" {
		return nil
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
