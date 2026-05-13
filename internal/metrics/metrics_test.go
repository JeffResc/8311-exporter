package metrics

import (
	"bytes"
	"testing"
)

func TestWriter_Basic(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	w.Header("ont_8311_rx_power_dbm", "Optical receive power in dBm.", Gauge)
	w.Sample("ont_8311_rx_power_dbm", -18.45)
	w.Header("ont_8311_cpu_temperature_celsius", "CPU temp.", Gauge)
	w.Sample("ont_8311_cpu_temperature_celsius", 66.65, "zone", "cpu1")
	w.Sample("ont_8311_cpu_temperature_celsius", 65.35, "zone", "cpu2")
	if err := w.Err(); err != nil {
		t.Fatalf("Err: %v", err)
	}

	want := "# HELP ont_8311_rx_power_dbm Optical receive power in dBm.\n" +
		"# TYPE ont_8311_rx_power_dbm gauge\n" +
		"ont_8311_rx_power_dbm -18.45\n" +
		"# HELP ont_8311_cpu_temperature_celsius CPU temp.\n" +
		"# TYPE ont_8311_cpu_temperature_celsius gauge\n" +
		"ont_8311_cpu_temperature_celsius{zone=\"cpu1\"} 66.65\n" +
		"ont_8311_cpu_temperature_celsius{zone=\"cpu2\"} 65.35\n"

	if got := buf.String(); got != want {
		t.Errorf("output mismatch\n--- got ---\n%s--- want ---\n%s", got, want)
	}
}

func TestWriter_EscapesLabelValue(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	w.Sample("m", 1, "k", `weird "value" \with\ newline`+"\n")
	want := `m{k="weird \"value\" \\with\\ newline\n"} 1` + "\n"
	if got := buf.String(); got != want {
		t.Errorf("escape mismatch\n got: %q\nwant: %q", got, want)
	}
}

func TestWriter_OddLabelCount(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	w.Sample("m", 1, "only-key")
	if !bytes.Contains(buf.Bytes(), []byte("ERROR odd label")) {
		t.Errorf("expected error marker, got %q", buf.String())
	}
}
