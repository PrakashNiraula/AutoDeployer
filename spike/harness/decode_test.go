package main

import (
	"bytes"
	"testing"

	"golang.org/x/text/encoding/japanese"
)

func TestLineDecoder_MultibyteSplitAcrossChunks(t *testing.T) {
	// "日" in Shift-JIS is 0x93 0xFA
	enc := japanese.ShiftJIS.NewEncoder()
	raw, err := enc.Bytes([]byte("日本語\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 3 {
		t.Fatalf("expected multibyte sequence, got %#v", raw)
	}

	d := NewLineDecoder("cp932")
	// Split in the middle of the first character.
	if _, err := d.Write(raw[:1]); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Write(raw[1:]); err != nil {
		t.Fatal(err)
	}
	got := d.String()
	if !bytes.Contains([]byte(got), []byte("日本語")) {
		t.Fatalf("mojibake or loss: got %q", got)
	}
}

func TestLineDecoder_UTF8(t *testing.T) {
	d := NewLineDecoder("utf-8")
	_, _ = d.Write([]byte("日本語テスト_utf8\n"))
	got := d.String()
	if got != "日本語テスト_utf8\n" && got != "日本語テスト_utf8" {
		if !bytes.Contains([]byte(got), []byte("日本語テスト_utf8")) {
			t.Fatalf("got %q", got)
		}
	}
}

func TestLineDecoder_IncompleteTrailAtEnd(t *testing.T) {
	d := NewLineDecoder("cp932")
	// Lone lead byte, no newline — Flush must not panic.
	_, _ = d.Write([]byte{0x93})
	_ = d.String()
}
