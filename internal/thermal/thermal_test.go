package thermal

import (
	"math"
	"path/filepath"
	"testing"
)

func TestZoneAt_ParityFixture(t *testing.T) {
	cases := []struct {
		file string
		want float64
	}{
		{"zone0.txt", 66.653},
		{"zone1.txt", 65.356},
	}
	for _, c := range cases {
		path := filepath.Join("..", "..", "testdata", "parity", c.file)
		got, err := ZoneAt(path)
		if err != nil {
			t.Fatalf("ZoneAt(%s): %v", c.file, err)
		}
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%s: got %v, want %v", c.file, got, c.want)
		}
	}
}
