// Package adtime provides the canonical AutoDeployer timestamp helpers (plan 4.2).
package adtime

import (
	"fmt"
	"strings"
	"time"
)

// Instant is a moment expressed in canonical local+offset form plus UTC parts.
type Instant struct {
	Canonical string // YYYY-MM-DDTHH:mm:ss±HH:MM
	UTC       string // RFC3339 UTC
	Local     string // YYYY-MM-DDTHH:mm:ss (no offset)
	Offset    string // ±HH:MM
	Time      time.Time
}

// Now returns Instant for the current local time.
func Now() Instant {
	return From(time.Now())
}

// From converts t into Instant using t's own location (not the host Local zone).
// Callers that want "machine local" should pass t.In(time.Local) or use Now().
func From(t time.Time) Instant {
	if t.Location() == nil {
		t = t.In(time.UTC)
	}
	_, offSec := t.Zone()
	sign := "+"
	if offSec < 0 {
		sign = "-"
		offSec = -offSec
	}
	hh := offSec / 3600
	mm := (offSec % 3600) / 60
	offset := fmt.Sprintf("%s%02d:%02d", sign, hh, mm)
	local := t.Format("2006-01-02T15:04:05")
	canonical := local + offset
	return Instant{
		Canonical: canonical,
		UTC:       t.UTC().Format(time.RFC3339),
		Local:     local,
		Offset:    offset,
		Time:      t,
	}
}

// In returns Instant for t interpreted in loc.
func In(t time.Time, loc *time.Location) Instant {
	if loc == nil {
		loc = time.UTC
	}
	return From(t.In(loc))
}

// ParseCanonical parses YYYY-MM-DDTHH:mm:ss±HH:MM.
func ParseCanonical(s string) (Instant, error) {
	t, err := time.Parse("2006-01-02T15:04:05-07:00", s)
	if err != nil {
		return Instant{}, fmt.Errorf("canonical time: %w", err)
	}
	return From(t), nil
}

// LogFileStamp returns YYYYMMDD_HHmmss_p0900 style stamp from Instant.
func (i Instant) LogFileStamp() string {
	t := i.Time
	base := t.Format("20060102_150405")
	off := i.Offset
	prefix := "p"
	if strings.HasPrefix(off, "-") {
		prefix = "m"
		off = strings.TrimPrefix(off, "-")
	} else {
		off = strings.TrimPrefix(off, "+")
	}
	off = strings.ReplaceAll(off, ":", "")
	return fmt.Sprintf("%s_%s%s", base, prefix, off)
}
