// Package ploam parses the output of `pon psg`, which reports the current
// PLOAM state machine position on the ONU.
//
// Example output:
//
//	errorcode=0 current=51 previous=40 time_curr=57779
package ploam

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// PonBinary is the path to the firmware's `pon` CLI on the device.
const PonBinary = "/usr/bin/pon"

// DefaultTimeout is a generous per-invocation deadline. `pon psg` returns
// in <10 ms in practice; the timeout protects against a hung binary
// blocking subsequent scrapes (which serialise behind a global mutex).
const DefaultTimeout = 5 * time.Second

// State is the parsed result of `pon psg`.
type State struct {
	Current     int
	Previous    int
	TimeCurrent int // seconds in current state
	ErrorCode   int
}

// Read invokes `pon psg` and parses the result.
func Read() (State, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, PonBinary, "psg").Output()
	if err != nil {
		return State{}, fmt.Errorf("exec pon psg: %w", err)
	}
	return Parse(string(out))
}

// Parse extracts the integer fields of a `pon psg` line. The output looks
// like `errorcode=0 current=51 previous=40 time_curr=57779`.
func Parse(s string) (State, error) {
	st := State{ErrorCode: -1}
	any := false
	for _, tok := range strings.Fields(s) {
		key, val, ok := strings.Cut(tok, "=")
		if !ok {
			continue
		}
		v, err := strconv.Atoi(val)
		if err != nil {
			continue
		}
		any = true
		switch key {
		case "errorcode":
			st.ErrorCode = v
		case "current":
			st.Current = v
		case "previous":
			st.Previous = v
		case "time_curr":
			st.TimeCurrent = v
		}
	}
	if !any {
		return st, fmt.Errorf("no key=value pairs in pon psg output")
	}
	return st, nil
}

// Name maps a numeric PLOAM state to its (state code, human description).
// Source: pon_state() in /usr/lib/lua/luci/controller/8311.lua plus the
// G.989.3 / XGS-PON state machine spec.
func Name(state int) (code, desc string) {
	if v, ok := names[state]; ok {
		return v.code, v.desc
	}
	return "Unknown", "Unknown PLOAM state"
}

type nameEntry struct{ code, desc string }

var names = map[int]nameEntry{
	0:  {"O0", "Power-up state"},
	10: {"O1", "Initial state"},
	11: {"O1.1", "Off-sync state"},
	12: {"O1.2", "Profile learning state"},
	20: {"O2", "Stand-by state"},
	23: {"O2.3", "Serial number state"},
	30: {"O3", "Serial number state"},
	40: {"O4", "Ranging state"},
	50: {"O5", "Operation state"},
	51: {"O5.1", "Associated state"},
	52: {"O5.2", "Pending state"},
	60: {"O6", "Intermittent LODS state"},
	70: {"O7", "Emergency stop state"},
	71: {"O7.1", "Emergency stop off-sync state"},
	72: {"O7.2", "Emergency stop in-sync state"},
	81: {"O8.1", "Downstream tuning off-sync state"},
	82: {"O8.2", "Downstream tuning profile learning state"},
	90: {"O9", "Upstream tuning state"},
}
