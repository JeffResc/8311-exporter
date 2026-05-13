package pontop

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "pontop", name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return b
}

func TestParse_KV_FEC(t *testing.T) {
	p, err := Parse(readFixture(t, "FEC_Status__Counters.txt"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if p.Mode != ModeKV {
		t.Fatalf("Mode: got %v want KV", p.Mode)
	}
	if p.Title != "FEC Status & Counters" {
		t.Errorf("Title: %q", p.Title)
	}
	want := map[string]string{
		"FEC upstream":            "ON",
		"FEC downstream":          "ON",
		"BIP errors":              "0",
		"Total FEC codewords":     "289895929785",
		"Corrected FEC codewords": "0",
		"FEC errored seconds":     "0",
	}
	for k, v := range want {
		if p.KV[k] != v {
			t.Errorf("KV[%q] = %q, want %q", k, p.KV[k], v)
		}
	}
}

func TestParse_KV_PLOAM_Downstream(t *testing.T) {
	p, err := Parse(readFixture(t, "PLOAM_Downstream_Counters.txt"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if p.KV["Burst profile"] != "346764 messages" {
		t.Errorf("got %q", p.KV["Burst profile"])
	}
	n, err := FirstInt(p.KV["Burst profile"])
	if err != nil || n != 346764 {
		t.Errorf("FirstInt: %d, err=%v", n, err)
	}
}

func TestParse_Table_Allocations(t *testing.T) {
	p, err := Parse(readFixture(t, "Allocation_Counters.txt"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if p.Mode != ModeTable {
		t.Fatalf("Mode: %v", p.Mode)
	}
	if len(p.Rows) != 2 {
		t.Fatalf("rows: %d", len(p.Rows))
	}
	first := p.Rows[0]
	if first["Alloc id"] != "15" {
		t.Errorf("first.Alloc id = %q", first["Alloc id"])
	}
	if first["Allocations"] != "57784120" {
		t.Errorf("first.Allocations = %q", first["Allocations"])
	}
	if first["Status"] != "3 LINKED" {
		t.Errorf("first.Status = %q", first["Status"])
	}
}

func TestParse_Table_GEMCounters(t *testing.T) {
	p, err := Parse(readFixture(t, "GEM_XGEM_Port_Counters.txt"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(p.Rows) != 5 {
		t.Fatalf("rows: %d", len(p.Rows))
	}
	// The third row is the main user GEM (id 1066) — sanity-check a few values.
	r := p.Rows[2]
	if r["GEM ID"] != "1066" {
		t.Errorf("GEM ID: %q", r["GEM ID"])
	}
	if r["u/s packets"] != "17329809" {
		t.Errorf("u/s packets: %q", r["u/s packets"])
	}
}

func TestParse_EmptyTable_Alarms(t *testing.T) {
	p, err := Parse(readFixture(t, "Active_alarms.txt"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if p.Mode != ModeTable {
		t.Fatalf("Mode: %v", p.Mode)
	}
	if len(p.Rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(p.Rows))
	}
	// Columns should still be detected.
	if len(p.Columns) == 0 {
		t.Error("expected non-empty Columns from header")
	}
}

func TestParse_Status(t *testing.T) {
	p, err := Parse(readFixture(t, "Status.txt"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if p.KV["PON IP SW version"] != "1.22.9" {
		t.Errorf("SW ver: %q", p.KV["PON IP SW version"])
	}
	if p.KV["ONU Authentication Status"] != "0" {
		t.Errorf("auth: %q", p.KV["ONU Authentication Status"])
	}
	if p.KV["FEC upstream"] != "ON" {
		t.Errorf("fec up: %q", p.KV["FEC upstream"])
	}
}

func TestFirstInt(t *testing.T) {
	cases := []struct {
		in   string
		want int64
		ok   bool
	}{
		{"346764 messages", 346764, true},
		{"53 deg C / 326 K", 53, true},
		{"-5", -5, true},
		{"0x00000731", 0x731, true},
		{"3 LINKED", 3, true},
		{"Not supported", 0, false},
		{"", 0, false},
	}
	for _, c := range cases {
		got, err := FirstInt(c.in)
		if c.ok && err != nil {
			t.Errorf("%q: unexpected err %v", c.in, err)
		}
		if !c.ok && err == nil {
			t.Errorf("%q: expected error", c.in)
		}
		if c.ok && got != c.want {
			t.Errorf("%q: got %d want %d", c.in, got, c.want)
		}
	}
}

func TestFirstFloat(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"3.43 V", 3.43},
		{"-18.45 dBm", -18.45},
		{"5.14 dBm", 5.14},
		{"12.86 mA", 12.86},
		{"53 deg C / 326 K", 53},
	}
	for _, c := range cases {
		got, err := FirstFloat(c.in)
		if err != nil {
			t.Errorf("%q: err %v", c.in, err)
			continue
		}
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%q: got %v want %v", c.in, got, c.want)
		}
	}
}

func TestBool(t *testing.T) {
	cases := []struct {
		in string
		v  float64
		ok bool
	}{
		{"ON", 1, true},
		{"OFF", 0, true},
		{"ENABLED ", 1, true},
		{"DISABLED", 0, true},
		{"value not available", 0, false},
	}
	for _, c := range cases {
		v, ok := Bool(c.in)
		if ok != c.ok || (ok && v != c.v) {
			t.Errorf("Bool(%q) = (%v,%v) want (%v,%v)", c.in, v, ok, c.v, c.ok)
		}
	}
}
