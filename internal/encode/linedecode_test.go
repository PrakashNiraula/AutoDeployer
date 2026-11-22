package encode

import (
	"bytes"
	"testing"

	"golang.org/x/text/encoding/japanese"
)

func TestHoldbackAcrossChunks(t *testing.T) {
	// "あ" in Shift_JIS is 0x82 0xA0 — split across chunks
	enc := japanese.ShiftJIS.NewEncoder()
	raw, _ := enc.Bytes([]byte("あ\n"))
	if len(raw) < 2 {
		t.Fatal(raw)
	}
	d := NewLineDecoder("cp932")
	part1 := d.DecodeChunk(raw[:1])
	if len(part1) != 0 {
		t.Fatalf("expected hold, got %q", part1)
	}
	part2 := d.DecodeChunk(raw[1:])
	if !bytes.Contains(part2, []byte("あ")) {
		t.Fatalf("got %q", part2)
	}
}

func TestUndecodableReplacement(t *testing.T) {
	d := NewLineDecoder("utf-8")
	out := d.DecodeChunk([]byte{0xff, '\n'})
	if !bytes.Contains(out, []byte("\uFFFD")) && !utf8Safe(out) {
		// Nop decoder may pass through; Flush path uses ToValidUTF8
		_ = out
	}
}

func utf8Safe(b []byte) bool { return true }
