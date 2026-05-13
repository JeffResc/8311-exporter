// Package sysfs contains tiny helpers for reading single-value files out of
// /sys (and other procfs-shaped paths). Most sysfs leaves are one line of
// text — the helpers trim the trailing newline and parse cheaply.
package sysfs

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ReadString returns the trimmed contents of path, or "" if the file can't
// be read. Use for attributes where "missing" is a normal outcome (e.g.
// not all netdevs expose every file).
func ReadString(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// ReadInt64 parses the file at path as a signed int64. Returns a wrapped
// error so callers can attach path context to diagnostics.
func ReadInt64(path string) (int64, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", path, err)
	}
	v, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", path, err)
	}
	return v, nil
}
