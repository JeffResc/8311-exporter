package ploam

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse_ParityFixture(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "parity", "psg.txt")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	st, err := Parse(string(b))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if st.ErrorCode != 0 {
		t.Errorf("ErrorCode: got %d want 0", st.ErrorCode)
	}
	if st.Current != 51 {
		t.Errorf("Current: got %d want 51", st.Current)
	}
	if st.Previous != 40 {
		t.Errorf("Previous: got %d want 40", st.Previous)
	}
	if st.TimeCurrent != 57779 {
		t.Errorf("TimeCurrent: got %d want 57779", st.TimeCurrent)
	}
}

func TestParse_Empty(t *testing.T) {
	if _, err := Parse(""); err == nil {
		t.Fatal("want error on empty input")
	}
}

func TestName(t *testing.T) {
	code, desc := Name(51)
	if code != "O5.1" {
		t.Errorf("code: got %q want O5.1", code)
	}
	if desc != "Associated state" {
		t.Errorf("desc: got %q", desc)
	}
	if c, _ := Name(9999); c != "Unknown" {
		t.Errorf("unknown state should return Unknown, got %q", c)
	}
}
