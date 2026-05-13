// Package exporter wires up the per-data-source readers into a single
// HTTP handler that emits Prometheus text-format metrics. Every metric is
// prefixed `ont_8311_`.
package exporter

import (
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/jeffresc/8311-exporter/internal/metrics"
	"github.com/jeffresc/8311-exporter/internal/optics"
	"github.com/jeffresc/8311-exporter/internal/ploam"
	"github.com/jeffresc/8311-exporter/internal/thermal"
)

// Namespace is the metric-name prefix applied to every series emitted by
// this exporter.
const Namespace = "ont_8311"

// Handler returns an http.Handler that emits the full metric set. The
// includeEthtool flag controls whether per-interface driver-private counters
// are collected via the ETHTOOL_GSTATS ioctl — adds ~100 ms to a ~2.1 s
// scrape on the 400 MHz SoC.
func Handler(includeEthtool bool) http.Handler {
	return &handler{includeEthtool: includeEthtool}
}

type handler struct {
	mu             sync.Mutex // serialises scrapes so concurrent requests can't pile on pontop
	includeEthtool bool
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	mw := metrics.NewWriter(w)

	collectOptics(mw)
	collectThermal(mw)
	collectPLOAM(mw)
	collectStatus(mw)
	collectOpticalStatus(mw)
	collectFEC(mw)
	collectGTC(mw)
	collectAlarms(mw)
	collectPLOAMMessages(mw)
	collectAllocations(mw)
	collectGEMPortCounters(mw)
	collectGEMPortStatus(mw)
	collectGEMEthCounters(mw)
	collectPowerSave(mw)
	collectPSM(mw)
	collectCapabilities(mw)
	collectOpticInfo(mw)
	collectInterfaces(mw, h.includeEthtool)
	collectSystem(mw)
	emitBuildInfo(mw)

	if err := mw.Err(); err != nil {
		log.Printf("write metrics: %v", err)
	}
}

// scrape times the body and emits ont_8311_scrape_{duration_seconds,success}
// for the named source. Always defer this.
func scrape(mw *metrics.Writer, source string, body func() error) {
	start := time.Now()
	err := body()
	dur := time.Since(start).Seconds()

	mw.Sample(Namespace+"_scrape_duration_seconds", dur, "source", source)
	if err == nil {
		mw.Sample(Namespace+"_scrape_success", 1, "source", source)
	} else {
		mw.Sample(Namespace+"_scrape_success", 0, "source", source)
		log.Printf("%s: %v", source, err)
	}
}

func collectOptics(mw *metrics.Writer) {
	// Write the family headers up front so the output is well-formed even if
	// the read fails (in which case no samples follow).
	mw.Header(Namespace+"_optic_temperature_celsius", "SFP module temperature in degrees Celsius (SFF-8472 DDM).", metrics.Gauge)
	mw.Header(Namespace+"_module_voltage_volts", "SFP module supply voltage in volts.", metrics.Gauge)
	mw.Header(Namespace+"_tx_bias_milliamperes", "Laser transmit bias current in milliamperes.", metrics.Gauge)
	mw.Header(Namespace+"_tx_power_dbm", "Optical transmit power in dBm.", metrics.Gauge)
	mw.Header(Namespace+"_rx_power_dbm", "Optical receive power in dBm.", metrics.Gauge)
	mw.Header(Namespace+"_tx_power_milliwatts", "Optical transmit power in milliwatts.", metrics.Gauge)
	mw.Header(Namespace+"_rx_power_milliwatts", "Optical receive power in milliwatts.", metrics.Gauge)

	scrape(mw, "optics", func() error {
		r, err := optics.Read()
		if err != nil {
			return err
		}
		mw.Sample(Namespace+"_optic_temperature_celsius", r.TempC)
		mw.Sample(Namespace+"_module_voltage_volts", r.VoltageV)
		mw.Sample(Namespace+"_tx_bias_milliamperes", r.TxBiasMA)
		mw.Sample(Namespace+"_tx_power_dbm", r.TxPowerDBm)
		mw.Sample(Namespace+"_rx_power_dbm", r.RxPowerDBm)
		mw.Sample(Namespace+"_tx_power_milliwatts", r.TxPowerMW)
		mw.Sample(Namespace+"_rx_power_milliwatts", r.RxPowerMW)
		return nil
	})
}

func collectThermal(mw *metrics.Writer) {
	mw.Header(Namespace+"_cpu_temperature_celsius", "CPU core temperature in degrees Celsius (from /sys/class/thermal).", metrics.Gauge)
	scrape(mw, "thermal", func() error {
		var firstErr error
		for i := 0; i < 2; i++ {
			t, err := thermal.Zone(i)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			mw.Sample(Namespace+"_cpu_temperature_celsius", t,
				"zone", "cpu"+strconv.Itoa(i+1))
		}
		return firstErr
	})
}

func collectPLOAM(mw *metrics.Writer) {
	mw.Header(Namespace+"_ploam_state", "Current PLOAM state machine value (numeric, e.g. 51 = O5.1 Associated).", metrics.Gauge)
	mw.Header(Namespace+"_ploam_previous_state", "Previous PLOAM state machine value.", metrics.Gauge)
	mw.Header(Namespace+"_ploam_time_in_state_seconds", "Seconds since entering the current PLOAM state.", metrics.Gauge)
	mw.Header(Namespace+"_ploam_errorcode", "Error code returned by `pon psg` (0 means OK).", metrics.Gauge)
	mw.Header(Namespace+"_ploam_state_info", "PLOAM state metadata. Value is 1; useful labels are code and description.", metrics.Gauge)

	scrape(mw, "ploam", func() error {
		st, err := ploam.Read()
		if err != nil {
			return err
		}
		mw.Sample(Namespace+"_ploam_state", float64(st.Current))
		mw.Sample(Namespace+"_ploam_previous_state", float64(st.Previous))
		mw.Sample(Namespace+"_ploam_time_in_state_seconds", float64(st.TimeCurrent))
		mw.Sample(Namespace+"_ploam_errorcode", float64(st.ErrorCode))
		code, desc := ploam.Name(st.Current)
		mw.Sample(Namespace+"_ploam_state_info", 1, "code", code, "description", desc)
		return nil
	})
}
