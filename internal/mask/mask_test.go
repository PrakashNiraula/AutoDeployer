package mask

import "testing"

func TestMaskMapAndShortValues(t *testing.T) {
	m := MaskMap(map[string]string{
		"DB_PASSWORD": "secretvalue",
		"SHORT":       "ab",
		"HOST":        "localhost",
	}, []string{"SHORT"}, 4)
	if m["DB_PASSWORD"] != "***" {
		t.Fatalf("auto mask password: %q", m["DB_PASSWORD"])
	}
	if m["SHORT"] != "ab" {
		t.Fatalf("short declared secret should not mask: %q", m["SHORT"])
	}
	if m["HOST"] != "localhost" {
		t.Fatalf("host changed")
	}
}

func TestApplyEscapesRegexChars(t *testing.T) {
	got := Apply("pw=a+b*c?", []string{"a+b*c?"}, 4)
	if got != "pw=***" {
		t.Fatalf("got %q", got)
	}
}
