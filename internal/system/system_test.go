package system

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseLoadAvg_Fixture(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "proc", "loadavg.txt"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	la, err := ParseLoadAvg(string(b))
	if err != nil {
		t.Fatalf("ParseLoadAvg: %v", err)
	}
	if la.One < 0 || la.Five < 0 || la.Fifteen < 0 {
		t.Errorf("negative load: %+v", la)
	}
	if la.Total <= 0 {
		t.Errorf("Total should be positive, got %d", la.Total)
	}
}

func TestParseLoadAvg_Explicit(t *testing.T) {
	la, err := ParseLoadAvg("0.07 0.16 0.18 1/106 23346")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if la.One != 0.07 || la.Five != 0.16 || la.Fifteen != 0.18 {
		t.Errorf("loads wrong: %+v", la)
	}
	if la.Runnable != 1 || la.Total != 106 {
		t.Errorf("runnable/total wrong: %+v", la)
	}
}

func TestParseUptime_Explicit(t *testing.T) {
	u, err := ParseUptime("1234.56 7890.12")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if u.UpSeconds != 1234.56 || u.IdleSeconds != 7890.12 {
		t.Errorf("uptime: %+v", u)
	}
}

func TestParseMemInfo_Fixture(t *testing.T) {
	f, err := os.Open(filepath.Join("..", "..", "testdata", "proc", "meminfo.txt"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	mi, err := ParseMemInfo(f)
	if err != nil {
		t.Fatalf("ParseMemInfo: %v", err)
	}
	if mi["MemTotal"] <= 0 {
		t.Errorf("MemTotal: %d", mi["MemTotal"])
	}
	if mi["MemFree"] <= 0 {
		t.Errorf("MemFree: %d", mi["MemFree"])
	}
	// MemTotal should be in bytes (kB * 1024), so > 100 MB on this device.
	if mi["MemTotal"] < 100*1024*1024 {
		t.Errorf("MemTotal looks too small (expected bytes): %d", mi["MemTotal"])
	}
}
