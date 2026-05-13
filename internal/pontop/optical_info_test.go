package pontop

import "testing"

// Optical Interface Info uses a non-standard column header
// ("SFP+ information   Status") instead of the usual "OPTION VALUE", but the
// body is KV-shaped. Verify the parser still treats it as KV.
func TestParse_KV_OpticalInfo(t *testing.T) {
	p, err := Parse(readFixture(t, "Optical_Interface_Info.txt"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if p.Mode != ModeKV {
		t.Fatalf("Mode: got %v want KV", p.Mode)
	}
	cases := map[string]string{
		"Vendor name": "OEM",
		"Part number": "SFP+ONU-XGSPON",
		"Revision":    "A-01",
		"Wavelength":  "1270 nm",
	}
	for k, want := range cases {
		if p.KV[k] != want {
			t.Errorf("KV[%q] = %q, want %q", k, p.KV[k], want)
		}
	}
}
