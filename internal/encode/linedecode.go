// Package encode decodes child-process output on line boundaries (plan 4.3).
package encode

import (
	"bytes"
	"io"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

// LineDecoder decodes bytes on line boundaries with multibyte holdback.
type LineDecoder struct {
	dec  *encoding.Decoder
	hold []byte
}

func NewLineDecoder(codePage string) *LineDecoder {
	return &LineDecoder{dec: decoderFor(codePage)}
}

func decoderFor(codePage string) *encoding.Decoder {
	switch strings.ToLower(strings.TrimSpace(codePage)) {
	case "utf-8", "utf8", "":
		return encoding.Nop.NewDecoder()
	case "cp932", "shift_jis", "sjis":
		return japanese.ShiftJIS.NewDecoder()
	case "cp1252":
		return charmap.Windows1252.NewDecoder()
	case "cp437":
		return charmap.CodePage437.NewDecoder()
	case "auto":
		return encoding.Nop.NewDecoder()
	default:
		return encoding.Nop.NewDecoder()
	}
}

// DecodeChunk returns complete decoded lines (including newlines) and holds a trailing partial line.
func (d *LineDecoder) DecodeChunk(p []byte) []byte {
	buf := append(append([]byte{}, d.hold...), p...)
	d.hold = nil
	var out bytes.Buffer
	for {
		i := bytes.IndexByte(buf, '\n')
		if i < 0 {
			d.hold = append([]byte{}, buf...)
			return out.Bytes()
		}
		line := buf[:i+1]
		buf = buf[i+1:]
		out.Write(d.decodeLine(line))
	}
}

// Flush emits any held bytes (undecodable → U+FFFD).
func (d *LineDecoder) Flush() []byte {
	if len(d.hold) == 0 {
		return nil
	}
	decoded := d.decodeLine(append(d.hold, '\n'))
	d.hold = nil
	return bytes.TrimSuffix(decoded, []byte{'\n'})
}

func (d *LineDecoder) decodeLine(line []byte) []byte {
	r := transform.NewReader(bytes.NewReader(line), d.dec)
	out, err := io.ReadAll(r)
	if err != nil {
		return bytes.ToValidUTF8(line, []byte("\uFFFD"))
	}
	if !utf8.Valid(out) {
		out = bytes.ToValidUTF8(out, []byte("\uFFFD"))
	}
	return out
}
