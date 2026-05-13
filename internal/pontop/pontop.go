// Package pontop parses the batch-mode output of the firmware's `pontop`
// diagnostic tool and exposes a tiny subprocess runner.
//
// `pontop -g "<page>" -b` writes plain-text dumps in one of two layouts:
//
//  1. Key/value pages — start with a line like `OPTION ... VALUE`, then
//     one record per line: `<key padded to col N> : <value with units>`.
//  2. Table pages — start with a column-header line (e.g.
//     `Alloc index   Alloc id   ...`), then data rows separated by
//     two-or-more spaces.
//
// We don't fight that: Parse returns the parsed payload as either a Page.KV
// or Page.Rows, and per-metric extraction lives in the caller.
package pontop

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Binary is the path to the pontop CLI on the device.
const Binary = "/usr/bin/pontop"

// runMu serialises subprocess launches. pontop reads from /dev/pon and the
// firmware's docs warn against concurrent invocations.
var runMu sync.Mutex

// Run invokes `pontop -g <page> -b` with a timeout and returns stdout.
func Run(ctx context.Context, page string) ([]byte, error) {
	runMu.Lock()
	defer runMu.Unlock()

	cmd := exec.CommandContext(ctx, Binary, "-g", page, "-b")
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return nil, fmt.Errorf("pontop %q exit %d: %s", page, ee.ExitCode(), strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, fmt.Errorf("pontop %q: %w", page, err)
	}
	return out, nil
}

// DefaultTimeout is a generous per-page deadline. In practice pontop returns
// within <100 ms; we allow a lot of headroom for slow CPU loads.
const DefaultTimeout = 5 * time.Second

// RunDefault wraps Run with DefaultTimeout.
func RunDefault(page string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	return Run(ctx, page)
}

// Page is the result of parsing one pontop batch dump.
type Page struct {
	Title string // e.g. "FEC Status & Counters" (from the first line, if present)
	Mode  Mode   // KV or Table

	// KV is populated when Mode == ModeKV. Iteration order is preserved
	// via Keys.
	KV   map[string]string
	Keys []string

	// Columns and Rows are populated when Mode == ModeTable.
	Columns []string
	Rows    []map[string]string
}

// Mode discriminates the two layouts pontop emits in batch mode.
type Mode int

// Page modes recognised by Parse.
const (
	ModeKV Mode = iota
	ModeTable
)

// Parse decodes batch-mode pontop output. It auto-detects KV vs table mode.
func Parse(b []byte) (Page, error) {
	p := Page{KV: map[string]string{}}

	// Strip a trailing NUL if any (pontop sometimes terminates with one) and
	// trim a UTF-8 BOM.
	b = trimRight(b)

	// Find the title and the header line.
	scanner := bufio.NewScanner(strings.NewReader(string(b)))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return p, err
	}

	// Optional first line: "Page: <title>"
	idx := 0
	if idx < len(lines) && strings.HasPrefix(lines[idx], "Page:") {
		p.Title = strings.TrimSpace(strings.TrimPrefix(lines[idx], "Page:"))
		idx++
	}

	// Find first non-empty line — that's the column header.
	for idx < len(lines) && strings.TrimSpace(lines[idx]) == "" {
		idx++
	}
	if idx >= len(lines) {
		return p, nil
	}

	header := lines[idx]
	body := lines[idx+1:]

	// Decide KV vs Table by inspecting the body for ` : ` separators.
	// Most pages either use them everywhere (KV) or not at all (Table).
	// We treat anything with at least one separator as KV — the parser
	// already ignores section-heading lines without separators, so loose
	// detection is safe.
	kv := false
	for _, ln := range body {
		if indexOfColonSep(ln) >= 0 {
			kv = true
			break
		}
	}

	if kv {
		p.Mode = ModeKV
		// The "header" line in KV pages (e.g. "OPTION VALUE" or
		// "SFP+ information Status") never has a colon, so passing it
		// through parseKV is a no-op.
		parseKV(append([]string{header}, body...), &p)
	} else {
		p.Mode = ModeTable
		parseTable(header, body, &p)
	}

	return p, nil
}

// parseKV interprets the body of a key/value page. The separator is " : "
// (colon flanked by spaces) — anything before it is key, anything after is
// value. Lines without a colon (separators like "---") or blank lines are
// skipped. Lines that look like section headings (no colon, plain text) are
// also skipped — extraction logic doesn't need them.
func parseKV(body []string, p *Page) {
	for _, ln := range body {
		t := strings.TrimSpace(ln)
		if t == "" {
			continue
		}
		// Section separators / headings without a colon.
		colon := indexOfColonSep(ln)
		if colon < 0 {
			continue
		}
		key := strings.TrimSpace(ln[:colon])
		val := strings.TrimSpace(ln[colon+1:])
		if key == "" {
			continue
		}
		if _, exists := p.KV[key]; !exists {
			p.Keys = append(p.Keys, key)
		}
		p.KV[key] = val
	}
}

// indexOfColonSep returns the index of the ` : ` separator in a KV line, or
// -1 if not found. We require at least one space *before* the colon so that
// values containing colons (e.g. URLs) don't trip us up.
func indexOfColonSep(s string) int {
	// Look for the first occurrence of " : " (space-colon-space), or " :"
	// at end-of-line.
	for i := 1; i < len(s)-1; i++ {
		if s[i] == ':' && s[i-1] == ' ' {
			// Either space after or end of string.
			if i+1 >= len(s) || s[i+1] == ' ' || s[i+1] == '\t' {
				return i
			}
		}
	}
	return -1
}

// parseTable interprets a column-oriented page. Column boundaries are
// detected from the header: a new column begins where two-or-more spaces
// are followed by a non-space character, which lets column names contain a
// single internal space (e.g. "Alloc id", "GEM ID", "u/s packets"). The
// final column extends to end of line.
func parseTable(header string, body []string, p *Page) {
	type span struct {
		name  string
		start int
	}
	var spans []span
	if header == "" {
		return
	}
	// Find token starts: first non-space at position 0, or non-space preceded
	// by >=2 spaces.
	i := 0
	for i < len(header) && header[i] == ' ' {
		i++
	}
	if i >= len(header) {
		return
	}
	tokStart := i
	for i < len(header) {
		// Look for run of 2+ spaces, then a non-space.
		if header[i] == ' ' {
			runStart := i
			for i < len(header) && header[i] == ' ' {
				i++
			}
			runLen := i - runStart
			if i < len(header) && runLen >= 2 {
				spans = append(spans, span{name: strings.TrimRight(header[tokStart:runStart], " "), start: tokStart})
				tokStart = i
			}
			// Single-space within a column name — keep scanning the same column.
		} else {
			i++
		}
	}
	spans = append(spans, span{name: strings.TrimRight(header[tokStart:], " "), start: tokStart})
	if len(spans) == 0 {
		return
	}

	p.Columns = make([]string, len(spans))
	for i, s := range spans {
		p.Columns[i] = s.name
	}

	for _, ln := range body {
		t := strings.TrimSpace(ln)
		if t == "" {
			continue
		}
		row := map[string]string{}
		// Pad short lines so slicing stays safe.
		for i, s := range spans {
			start := s.start
			if start >= len(ln) {
				row[s.name] = ""
				continue
			}
			end := len(ln)
			if i+1 < len(spans) {
				end = spans[i+1].start
				if end > len(ln) {
					end = len(ln)
				}
			}
			row[s.name] = strings.TrimSpace(ln[start:end])
		}
		p.Rows = append(p.Rows, row)
	}
}

func trimRight(b []byte) []byte {
	for len(b) > 0 {
		last := b[len(b)-1]
		if last == 0 || last == '\n' || last == '\r' || last == ' ' || last == '\t' {
			b = b[:len(b)-1]
			continue
		}
		break
	}
	return b
}

// FirstInt extracts the leading signed integer from a value string. It is
// permissive: e.g. "346764 messages" -> 346764, "53 deg C / 326 K" -> 53,
// "0x00000000" -> 0 (decimal), "Not supported" -> error.
func FirstInt(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty value")
	}
	// Allow hex literals.
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		end := 2
		for end < len(s) && isHex(s[end]) {
			end++
		}
		return strconv.ParseInt(s[2:end], 16, 64)
	}
	start := 0
	if s[0] == '-' || s[0] == '+' {
		start = 1
	}
	end := start
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	if end == start {
		return 0, fmt.Errorf("no leading integer in %q", s)
	}
	return strconv.ParseInt(s[:end], 10, 64)
}

// FirstFloat extracts the leading float from a value string. Stops at the
// first space or non-numeric character after at least one digit.
func FirstFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	end := 0
	if end < len(s) && (s[end] == '-' || s[end] == '+') {
		end++
	}
	sawDigit := false
	sawDot := false
	for end < len(s) {
		c := s[end]
		if c >= '0' && c <= '9' {
			sawDigit = true
			end++
			continue
		}
		if c == '.' && !sawDot {
			sawDot = true
			end++
			continue
		}
		break
	}
	if !sawDigit {
		return 0, fmt.Errorf("no leading float in %q", s)
	}
	return strconv.ParseFloat(s[:end], 64)
}

// Bool maps the common pontop status words to 0/1. Recognised:
//
//	ON / OFF, ENABLED / DISABLED, OK / ERROR, YES / NO, TRUE / FALSE.
//
// Returns (value, true) on hit, (0, false) on miss.
func Bool(s string) (float64, bool) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "ON", "ENABLED", "OK", "YES", "TRUE", "1":
		return 1, true
	case "OFF", "DISABLED", "ERROR", "NO", "FALSE", "0":
		return 0, true
	}
	return 0, false
}

func isHex(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}
