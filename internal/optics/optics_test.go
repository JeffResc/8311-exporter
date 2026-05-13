package optics

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

// Captured live values from the parity fixture (testdata/parity/eeprom51.bin):
//
//	bytes 96..107 = 35 22 85 9e 19 8f 82 26 00 8f e3 29
//
// Expected (per the Lua formula in /usr/lib/lua/8311/tools.lua):
//
//	optic_tempC    = 0x35 + 0x22/256        = 53 + 0.1328125    = 53.1328125
//	voltage        = 0x859e / 10000         = 34206 / 10000     = 3.4206
//	tx_bias_mA     = 0x198f / 500           = 6543 / 500         = 13.086
//	tx_power_mW    = 0x8226 / 10000         = 33318 / 10000      = 3.3318
//	rx_power_mW    = 0x008f / 10000         = 143 / 10000        = 0.0143
//	tx_power_dBm   = 10*log10(3.3318)       ≈ 5.226789
//	rx_power_dBm   = 10*log10(0.0143)       ≈ -18.446640
func TestParse_ParityFixture(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "parity", "eeprom51.bin")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	got, err := Parse(b)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	cases := []struct {
		name string
		got  float64
		want float64
		tol  float64
	}{
		{"TempC", got.TempC, 53.1328125, 1e-9},
		{"VoltageV", got.VoltageV, 3.4206, 1e-9},
		{"TxBiasMA", got.TxBiasMA, 13.086, 1e-9},
		{"TxPowerMW", got.TxPowerMW, 3.3318, 1e-9},
		{"RxPowerMW", got.RxPowerMW, 0.0143, 1e-9},
		{"TxPowerDBm", got.TxPowerDBm, 5.226789238562102, 1e-9},
		{"RxPowerDBm", got.RxPowerDBm, -18.44663962534938, 1e-9},
	}
	for _, c := range cases {
		if math.Abs(c.got-c.want) > c.tol {
			t.Errorf("%s: got %.9f, want %.9f (tol %g)", c.name, c.got, c.want, c.tol)
		}
	}
}

func TestParse_TooShort(t *testing.T) {
	if _, err := Parse(make([]byte, 50)); err == nil {
		t.Fatal("want error on short input")
	}
}

func TestParse_ZeroPower(t *testing.T) {
	b := make([]byte, MinBytes)
	// rx_mw = 0 -> rx_dBm should be -Inf, not NaN
	got, err := Parse(b)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !math.IsInf(got.RxPowerDBm, -1) {
		t.Errorf("RxPowerDBm: want -Inf for zero mW, got %v", got.RxPowerDBm)
	}
}
