// Package metrics emits Prometheus text-exposition format directly.
//
// Why not prometheus/client_golang? On a 400 MHz MIPS SoC with 1 GB of RAM
// the standard library is a meaningful chunk of resident memory and protobuf
// serialisation is wasted work — we always render text. A single-file
// formatter keeps the binary small (~2 MB stripped) and per-scrape
// allocations near zero.
//
// Format reference:
// https://github.com/prometheus/docs/blob/main/docs/instrumenting/exposition_formats.md
package metrics

import (
	"io"
	"strconv"
	"strings"
)

// Type is the Prometheus metric type identifier emitted in TYPE lines.
type Type string

// Metric types supported by this writer.
const (
	Gauge   Type = "gauge"
	Counter Type = "counter"
)

// Writer streams text-format metrics to an io.Writer. Not safe for concurrent
// use by multiple goroutines; callers must serialise.
type Writer struct {
	w   io.Writer
	err error
	buf []byte // re-used per metric line to keep alloc pressure low
}

// NewWriter returns a Writer that streams text-format metrics to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w, buf: make([]byte, 0, 256)}
}

// Err returns the first write error encountered, if any.
func (w *Writer) Err() error { return w.err }

func (w *Writer) write(b []byte) {
	if w.err != nil {
		return
	}
	_, w.err = w.w.Write(b)
}

func (w *Writer) writeString(s string) {
	if w.err != nil {
		return
	}
	_, w.err = io.WriteString(w.w, s)
}

// Header writes the HELP and TYPE lines for a metric family. Call once per
// metric name before emitting samples.
func (w *Writer) Header(name, help string, t Type) {
	w.writeString("# HELP ")
	w.writeString(name)
	w.writeString(" ")
	w.writeString(escapeHelp(help))
	w.writeString("\n# TYPE ")
	w.writeString(name)
	w.writeString(" ")
	w.writeString(string(t))
	w.writeString("\n")
}

// Sample emits one observation. labels may be nil. The first variadic pair is
// labelName, labelValue, repeated.
func (w *Writer) Sample(name string, value float64, labels ...string) {
	if len(labels)%2 != 0 {
		// Programmer error; emit so it's caught loudly in tests.
		w.writeString("# ERROR odd label count for ")
		w.writeString(name)
		w.writeString("\n")
		return
	}
	w.buf = w.buf[:0]
	w.buf = append(w.buf, name...)
	if len(labels) > 0 {
		w.buf = append(w.buf, '{')
		for i := 0; i < len(labels); i += 2 {
			if i > 0 {
				w.buf = append(w.buf, ',')
			}
			w.buf = append(w.buf, labels[i]...)
			w.buf = append(w.buf, '=', '"')
			w.buf = appendEscape(w.buf, labels[i+1])
			w.buf = append(w.buf, '"')
		}
		w.buf = append(w.buf, '}')
	}
	w.buf = append(w.buf, ' ')
	w.buf = strconv.AppendFloat(w.buf, value, 'g', -1, 64)
	w.buf = append(w.buf, '\n')
	w.write(w.buf)
}

// escapeHelp escapes the HELP text per the exposition format (backslash and
// newline).
func escapeHelp(s string) string {
	if !strings.ContainsAny(s, "\\\n") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// appendEscape escapes a label value per the exposition format
// (backslash, double-quote, newline).
func appendEscape(dst []byte, s string) []byte {
	if !strings.ContainsAny(s, "\\\"\n") {
		return append(dst, s...)
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\':
			dst = append(dst, '\\', '\\')
		case '"':
			dst = append(dst, '\\', '"')
		case '\n':
			dst = append(dst, '\\', 'n')
		default:
			dst = append(dst, c)
		}
	}
	return dst
}
