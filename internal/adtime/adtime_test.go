package adtime

import (
	"strings"
	"testing"
	"time"
)

func TestFromAndLogStampJST(t *testing.T) {
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

func TestFromUTC(t *testing.T) {
	tm := time.Date(2026, 11, 19, 17, 0, 0, 0, time.UTC)
	inst := From(tm)
	if inst.Canonical != "2026-11-19T17:00:00+00:00" {
		t.Fatalf("canonical=%s", inst.Canonical)
	}
	if stamp := inst.LogFileStamp(); stamp != "20261119_170000_p0000" {
		t.Fatalf("stamp=%s", stamp)
	}
}

func TestInInjectsLocation(t *testing.T) {
	// Absolute UTC instant viewed in Tokyo vs UTC must keep the same UTC column.
	utc := time.Date(2026, 11, 19, 17, 0, 0, 0, time.UTC)
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}
	a := In(utc, time.UTC)
	b := In(utc, tokyo)
	if a.UTC != b.UTC {
		t.Fatalf("UTC drift: %s vs %s", a.UTC, b.UTC)
	}
	if a.Canonical == b.Canonical {
		t.Fatalf("expected different local+offset, both %s", a.Canonical)
	}
	if b.Offset != "+09:00" {
		t.Fatalf("tokyo offset=%s", b.Offset)
	}
}

func TestHistoryOrderByUTCAcrossOffsets(t *testing.T) {
	// Two locals in different offsets that share the same UTC instant must
	// sort equal by UTC; an earlier UTC must sort first regardless of wall clock.
	tokyo := time.FixedZone("JST", 9*3600)
	ny := time.FixedZone("EST", -5*3600)
	earlier := From(time.Date(2026, 11, 20, 1, 0, 0, 0, tokyo))  // 2026-11-19T16:00:00Z
	later := From(time.Date(2026, 11, 19, 12, 0, 0, 0, ny))      // 2026-11-19T17:00:00Z
	if earlier.UTC >= later.UTC {
		t.Fatalf("UTC order wrong: earlier=%s later=%s", earlier.UTC, later.UTC)
	}
	sameA := From(time.Date(2026, 11, 20, 2, 0, 0, 0, tokyo))
	sameB := From(time.Date(2026, 11, 19, 12, 0, 0, 0, ny))
	if sameA.UTC != sameB.UTC {
		t.Fatalf("same instant UTC mismatch: %s vs %s", sameA.UTC, sameB.UTC)
	}
}

func TestParseCanonicalPreservesOffset(t *testing.T) {
	inst, err := ParseCanonical("2026-11-20T02:00:00+09:00")
	if err != nil {
		t.Fatal(err)
	}
	if inst.Canonical != "2026-11-20T02:00:00+09:00" {
		t.Fatalf("canonical=%s", inst.Canonical)
	}
}
