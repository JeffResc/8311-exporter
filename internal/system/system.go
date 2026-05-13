// Package system parses /proc/{loadavg,uptime,meminfo,stat} into typed values.
// All fields are plain numbers; the caller decides how to project them as
// Prometheus metrics.
package system

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// LoadAvg is the parsed contents of /proc/loadavg.
type LoadAvg struct {
	One, Five, Fifteen float64
	Runnable, Total    int64
}

// ReadLoadAvg parses /proc/loadavg.
func ReadLoadAvg() (LoadAvg, error) {
	b, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return LoadAvg{}, err
	}
	return ParseLoadAvg(string(b))
}

// ParseLoadAvg decodes a /proc/loadavg-formatted string.
func ParseLoadAvg(s string) (LoadAvg, error) {
	fields := strings.Fields(s)
	if len(fields) < 4 {
		return LoadAvg{}, fmt.Errorf("loadavg: short input %q", s)
	}
	la := LoadAvg{}
	var err error
	if la.One, err = strconv.ParseFloat(fields[0], 64); err != nil {
		return la, err
	}
	if la.Five, err = strconv.ParseFloat(fields[1], 64); err != nil {
		return la, err
	}
	if la.Fifteen, err = strconv.ParseFloat(fields[2], 64); err != nil {
		return la, err
	}
	// fields[3] is "runnable/total"
	parts := strings.SplitN(fields[3], "/", 2)
	if len(parts) == 2 {
		la.Runnable, _ = strconv.ParseInt(parts[0], 10, 64)
		la.Total, _ = strconv.ParseInt(parts[1], 10, 64)
	}
	return la, nil
}

// Uptime is the parsed contents of /proc/uptime.
type Uptime struct {
	UpSeconds   float64
	IdleSeconds float64
}

// ReadUptime parses /proc/uptime.
func ReadUptime() (Uptime, error) {
	b, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return Uptime{}, err
	}
	return ParseUptime(string(b))
}

// ParseUptime decodes a /proc/uptime-formatted string.
func ParseUptime(s string) (Uptime, error) {
	f := strings.Fields(s)
	if len(f) < 1 {
		return Uptime{}, fmt.Errorf("uptime: short input %q", s)
	}
	up, err := strconv.ParseFloat(f[0], 64)
	if err != nil {
		return Uptime{}, err
	}
	u := Uptime{UpSeconds: up}
	if len(f) >= 2 {
		u.IdleSeconds, _ = strconv.ParseFloat(f[1], 64)
	}
	return u, nil
}

// MemInfo maps every key in /proc/meminfo to bytes (the file reports kB; we
// multiply by 1024 here so callers see SI units).
type MemInfo map[string]int64

// ReadMemInfo parses /proc/meminfo.
func ReadMemInfo() (MemInfo, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return ParseMemInfo(f)
}

// ParseMemInfo decodes /proc/meminfo content from r.
func ParseMemInfo(r io.Reader) (MemInfo, error) {
	mi := MemInfo{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		key, rest, ok := strings.Cut(scanner.Text(), ":")
		if !ok || key == "" {
			continue
		}
		// Most entries look like "1234 kB"; some are bare numbers.
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			continue
		}
		n, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			continue
		}
		if len(fields) >= 2 && strings.EqualFold(fields[1], "kB") {
			n *= 1024
		}
		mi[key] = n
	}
	return mi, scanner.Err()
}
