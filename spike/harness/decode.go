package main

import (
	"bytes"
	"io"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

// LineDecoder decodes child-process bytes on line boundaries so a multibyte
// lead byte split across read chunks is never garbled. Incomplete trail bytes
// are held across Read calls; at Close/Flush remaining bytes become U+FFFD.
type LineDecoder struct {
	dec    *encoding.Decoder
	hold   []byte
	outBuf bytes.Buffer
}

func NewLineDecoder(codePage string) *LineDecoder {
	return &LineDecoder{dec: decoderFor(codePage)}
}

func decoderFor(codePage string) *encoding.Decoder {
	switch codePage {
	case "utf-8", "utf8", "UTF-8":
		return encoding.Nop.NewDecoder()
	case "cp932", "shift_jis", "sjis":
		return japanese.ShiftJIS.NewDecoder()
	case "cp1252":
		return charmap.Windows1252.NewDecoder()
	case "cp437":
		return charmap.CodePage437.NewDecoder()
	default:
		// "auto" and unknowns: treat as UTF-8 for harness (machine default is OS-specific).
		return encoding.Nop.NewDecoder()
	}
}

// Write accepts a chunk from the process pipe.
func (d *LineDecoder) Write(p []byte) (int, error) {
	buf := append(append([]byte{}, d.hold...), p...)
	d.hold = nil

	for {
		i := bytes.IndexByte(buf, '\n')
		if i < 0 {
			d.hold = append([]byte{}, buf...)
			return len(p), nil
		}
		line := buf[:i+1]
		buf = buf[i+1:]
		decoded, err := d.decodeLine(line)
		if err != nil {
			return len(p), err
		}
		d.outBuf.Write(decoded)
	}
}

func (d *LineDecoder) decodeLine(line []byte) ([]byte, error) {
	r := transform.NewReader(bytes.NewReader(line), d.dec)
	out, err := io.ReadAll(r)
	if err != nil {
		// Replace undecodable: fall back to lossy replacement via UTF-8 sanitize.
		return []byte(string(bytes.ToValidUTF8(line, []byte("\uFFFD")))), nil
	}
	if !utf8.Valid(out) {
		out = bytes.ToValidUTF8(out, []byte("\uFFFD"))
	}
	return out, nil
}

// Flush emits any held incomplete trailing bytes as replacement characters.
func (d *LineDecoder) Flush() []byte {
	if len(d.hold) == 0 {
		return d.outBuf.Bytes()
	}
	decoded, _ := d.decodeLine(append(d.hold, '\n'))
	d.hold = nil
	// strip the artificial newline we added for decode
	decoded = bytes.TrimSuffix(decoded, []byte{'\n'})
	d.outBuf.Write(decoded)
	return d.outBuf.Bytes()
}

func (d *LineDecoder) String() string {
	return string(d.Flush())
}
