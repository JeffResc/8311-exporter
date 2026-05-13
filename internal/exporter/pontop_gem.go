package exporter

import (
	"github.com/jeffresc/8311-exporter/internal/metrics"
	"github.com/jeffresc/8311-exporter/internal/pontop"
)

// collectGEMPortCounters reads `pontop "GEM/XGEM Port Counters"` and emits
// per-GEM packet/byte counters plus key-error counts.
func collectGEMPortCounters(mw *metrics.Writer) {
	mw.Header(Namespace+"_gem_packets_total", "Packets observed per GEM port, by direction.", metrics.Counter)
	mw.Header(Namespace+"_gem_bytes_total", "Bytes observed per GEM port, by direction.", metrics.Counter)
	mw.Header(Namespace+"_gem_key_errors_total", "Encryption key errors observed per GEM port.", metrics.Counter)

	scrape(mw, "pontop_gem_counters", func() error {
		return runPage("GEM/XGEM Port Counters", func(p pontop.Page) {
			for _, r := range p.Rows {
				id := r["GEM ID"]
				if id == "" {
					continue
				}
				emitInt(mw, Namespace+"_gem_packets_total", r["u/s packets"], "gem_id", id, "direction", "upstream")
				emitInt(mw, Namespace+"_gem_packets_total", r["d/s packets"], "gem_id", id, "direction", "downstream")
				emitInt(mw, Namespace+"_gem_bytes_total", r["u/s bytes"], "gem_id", id, "direction", "upstream")
				emitInt(mw, Namespace+"_gem_bytes_total", r["d/s bytes"], "gem_id", id, "direction", "downstream")
				emitInt(mw, Namespace+"_gem_key_errors_total", r["Key Errors"], "gem_id", id)
			}
		})
	})
}

func collectGEMPortStatus(mw *metrics.Writer) {
	mw.Header(Namespace+"_gem_max_size_bytes", "Configured maximum payload size per GEM port.", metrics.Gauge)
	mw.Header(Namespace+"_gem_port_info", "Per-GEM port configuration; value is always 1.", metrics.Gauge)

	scrape(mw, "pontop_gem_status", func() error {
		return runPage("GEM/XGEM Port Status", func(p pontop.Page) {
			for _, r := range p.Rows {
				id := r["GEM ID"]
				if id == "" {
					continue
				}
				emitInt(mw, Namespace+"_gem_max_size_bytes", r["Max. Size"], "gem_id", id)
				mw.Sample(Namespace+"_gem_port_info", 1,
					"gem_id", id,
					"alloc_id", r["Alloc Id"],
					"alloc_id_state", r["Alloc Id st."],
					"traffic_type", r["Data/OMCI"],
					"encryption", r["Encryption k.r."],
					"direction", r["Direction"],
				)
			}
		})
	})
}

// collectGEMEthCounters reads the Ethernet-layer packet-size histograms per
// GEM in both directions and emits them as Prometheus _bucket-style counters
// keyed by the size range.
func collectGEMEthCounters(mw *metrics.Writer) {
	mw.Header(Namespace+"_gem_eth_packets_total", "Per-GEM Ethernet packet counts bucketed by size, by direction.", metrics.Counter)

	for _, dir := range []struct{ page, label string }{
		{"GEM/XGEM port Eth DS Cnts", "downstream"},
		{"GEM/XGEM port Eth US Cnts", "upstream"},
	} {
		dir := dir
		scrape(mw, "pontop_gem_eth_"+dir.label, func() error {
			return runPage(dir.page, func(p pontop.Page) {
				// Columns: GEM Index, GEM ID, bytes <64, <128, <512, <=1518, >1518.
				// We expose every column whose header starts with '<' or '>' or '<='.
				for _, r := range p.Rows {
					id := r["GEM ID"]
					if id == "" {
						continue
					}
					for _, col := range p.Columns {
						switch col {
						case "GEM Index", "GEM ID":
							continue
						}
						emitInt(mw, Namespace+"_gem_eth_packets_total", r[col],
							"gem_id", id,
							"direction", dir.label,
							"size", col,
						)
					}
				}
			})
		})
	}
}

func collectPowerSave(mw *metrics.Writer) {
	mw.Header(Namespace+"_power_save_time_microseconds", "Time spent in each power-saving substate (µs).", metrics.Counter)
	mw.Header(Namespace+"_power_save_state_info", "Current power-saving state; value is 1.", metrics.Gauge)

	scrape(mw, "pontop_power_save", func() error {
		return runPage("Power Save Status", func(p pontop.Page) {
			mw.Sample(Namespace+"_power_save_state_info", 1, "state", p.KV["Power Save State"])
			emitInt(mw, Namespace+"_power_save_time_microseconds", p.KV["Total time"], "state", "total")
			emitInt(mw, Namespace+"_power_save_time_microseconds", p.KV["Doze time"], "state", "doze")
			emitInt(mw, Namespace+"_power_save_time_microseconds", p.KV["Cyclic sleep time"], "state", "cyclic_sleep")
			emitInt(mw, Namespace+"_power_save_time_microseconds", p.KV["Watchful sleep time"], "state", "watchful_sleep")
		})
	})
}

func collectPSM(mw *metrics.Writer) {
	mw.Header(Namespace+"_psm_enabled", "Whether PSM (power saving mode) is enabled.", metrics.Gauge)
	mw.Header(Namespace+"_psm_interval", "PSM-related configured intervals.", metrics.Gauge)
	mw.Header(Namespace+"_psm_state_total", "Cumulative PSM state-machine counter, by state.", metrics.Counter)

	scrape(mw, "pontop_psm", func() error {
		return runPage("PSM Configuration", func(p pontop.Page) {
			emitBool(mw, Namespace+"_psm_enabled", p.KV["PSM"])
			emitInt(mw, Namespace+"_psm_interval", p.KV["Maximum sleep interval"], "kind", "max_sleep")
			emitInt(mw, Namespace+"_psm_interval", p.KV["Minimum aware interval"], "kind", "min_aware")
			emitInt(mw, Namespace+"_psm_interval", p.KV["Minimum active held interval"], "kind", "min_active_held")
			emitInt(mw, Namespace+"_psm_interval", p.KV["Maximum Rx off interval"], "kind", "max_rx_off")

			states := []struct{ key, label string }{
				{"State idle", "idle"},
				{"State active", "active"},
				{"State active held", "active_held"},
				{"State active free", "active_free"},
				{"State asleep", "asleep"},
				{"State listen", "listen"},
			}
			for _, s := range states {
				emitInt(mw, Namespace+"_psm_state_total", p.KV[s.key], "state", s.label)
			}
		})
	})
}

func collectCapabilities(mw *metrics.Writer) {
	mw.Header(Namespace+"_capabilities_info", "ONU capabilities (modes, OMCI variants, etc). Value is 1.", metrics.Gauge)
	mw.Header(Namespace+"_capability_count", "ONU capability counts (e.g. GEM ports, allocations).", metrics.Gauge)

	scrape(mw, "pontop_capability", func() error {
		return runPage("Capability and Configuration", func(p pontop.Page) {
			mw.Sample(Namespace+"_capabilities_info", 1,
				"serial_number", p.KV["Serial number"],
				"basic_modes", p.KV["Basic mode(s)"],
				"omci_support", p.KV["OMCI support"],
				"power_saving_modes", p.KV["Power saving mode(s)"],
				"dba_modes", p.KV["DBA mode(s)"],
				"crypto_modes", p.KV["Crypto mode(s)"],
			)
			emitInt(mw, Namespace+"_capability_count", p.KV["GEM Ports"], "resource", "gem_ports")
			emitInt(mw, Namespace+"_capability_count", p.KV["Allocations"], "resource", "allocations")
		})
	})
}

func collectOpticInfo(mw *metrics.Writer) {
	mw.Header(Namespace+"_optic_info", "SFP module vendor information. Value is 1.", metrics.Gauge)
	mw.Header(Namespace+"_optic_wavelength_nanometers", "SFP module reported wavelength (nm).", metrics.Gauge)

	scrape(mw, "pontop_optic_info", func() error {
		return runPage("Optical Interface Info", func(p pontop.Page) {
			mw.Sample(Namespace+"_optic_info", 1,
				"vendor", p.KV["Vendor name"],
				"vendor_oui", p.KV["Vendor oui"],
				"part_number", p.KV["Part number"],
				"revision", p.KV["Revision"],
				"serial_number", p.KV["Serial number"],
				"date_code", p.KV["Date code"],
			)
			emitInt(mw, Namespace+"_optic_wavelength_nanometers", p.KV["Wavelength"])
		})
	})
}
