package adtime

import (
	"strings"
	"testing"
	"time"
)

func TestFromAndLogStamp(t *testing.T) {
	loc := time.FixedZone("JST", 9*3600)
	tm := time.Date(2026, 11, 20, 2, 0, 0, 0, loc)
	inst := From(tm)
	if inst.Canonical != "2026-11-20T02:00:00+09:00" {
		t.Fatalf("canonical=%s", inst.Canonical)
	}
	if !strings.HasPrefix(inst.UTC, "2026-11-19T17:00:00") {
		t.Fatalf("utc=%s", inst.UTC)
	}
	stamp := inst.LogFileStamp()
	if stamp != "20261120_020000_p0900" {
		t.Fatalf("stamp=%s", stamp)
	}
}
