package exporter

import (
	"fmt"
	"strings"

	"github.com/jeffresc/8311-exporter/internal/metrics"
	"github.com/jeffresc/8311-exporter/internal/pontop"
)

// runPage runs `pontop -g <page> -b`, parses it, and invokes emit with the
// parsed page. Errors are wrapped with the page name and returned so the
// caller can record scrape_success.
func runPage(page string, emit func(p pontop.Page)) error {
	out, err := pontop.RunDefault(page)
	if err != nil {
		return err
	}
	parsed, err := pontop.Parse(out)
	if err != nil {
		return fmt.Errorf("parse %q: %w", page, err)
	}
	emit(parsed)
	return nil
}

// snakeCase turns "Assign ONU ID" -> "assign_onu_id", "u/s packets" ->
// "us_packets", "GEM/XGEM port DS Counters" -> "gem_xgem_port_ds_counters".
// Used to derive Prometheus-safe labels from human-readable pontop keys.
func snakeCase(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevUnderscore := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z':
			b.WriteByte(c + 32)
			prevUnderscore = false
		case (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9'):
			b.WriteByte(c)
			prevUnderscore = false
		default:
			if !prevUnderscore {
				b.WriteByte('_')
				prevUnderscore = true
			}
		}
	}
	out := b.String()
	out = strings.Trim(out, "_")
	return out
}

func collectStatus(mw *metrics.Writer) {
	mw.Header(Namespace+"_info", "PON IP build and version information. Value is always 1.", metrics.Gauge)
	mw.Header(Namespace+"_fec_enabled", "Whether FEC is enabled in a given direction.", metrics.Gauge)
	mw.Header(Namespace+"_onu_authentication_status", "ONU authentication status as reported by pontop.", metrics.Gauge)

	scrape(mw, "pontop_status", func() error {
		return runPage("Status", func(p pontop.Page) {
			labels := []string{
				"hw_version", p.KV["PON IP HW version"],
				"fw_version", p.KV["PON IP FW version"],
				"sw_version", p.KV["PON IP SW version"],
				"pontop_version", p.KV["PON IP pontop version"],
				"pon_id", p.KV["PON ID"],
				"odn_class", p.KV["ODN Class"],
				"link_type", p.KV["Link Type"],
				"ploam_status", p.KV["PON PLOAM Status"],
			}
			mw.Sample(Namespace+"_info", 1, labels...)

			emitBool(mw, Namespace+"_fec_enabled", p.KV["FEC upstream"], "direction", "upstream")
			emitBool(mw, Namespace+"_fec_enabled", p.KV["FEC downstream"], "direction", "downstream")
			if n, err := pontop.FirstInt(p.KV["ONU Authentication Status"]); err == nil {
				mw.Sample(Namespace+"_onu_authentication_status", float64(n))
			}
		})
	})
}

func collectOpticalStatus(mw *metrics.Writer) {
	mw.Header(Namespace+"_optical_receiver_ok", "Whether the optical receiver reports OK status.", metrics.Gauge)
	mw.Header(Namespace+"_optical_transmitter_enabled", "Whether the optical transmitter is enabled.", metrics.Gauge)

	scrape(mw, "pontop_optical", func() error {
		return runPage("Optical Interface Status", func(p pontop.Page) {
			emitBool(mw, Namespace+"_optical_receiver_ok", p.KV["Receiver status"])
			emitBool(mw, Namespace+"_optical_transmitter_enabled", p.KV["Transmitter status"])
		})
	})
}

func collectFEC(mw *metrics.Writer) {
	// We keep the FEC enabled flag in collectStatus to avoid duplication.
	mw.Header(Namespace+"_fec_bip_errors_total", "Bit Interleaved Parity errors reported by FEC.", metrics.Counter)
	mw.Header(Namespace+"_fec_codewords_total", "FEC codewords seen, by outcome.", metrics.Counter)
	mw.Header(Namespace+"_fec_corrected_bytes_total", "Bytes corrected by FEC.", metrics.Counter)
	mw.Header(Namespace+"_fec_errored_seconds_total", "Seconds in which at least one FEC error occurred.", metrics.Counter)

	scrape(mw, "pontop_fec", func() error {
		return runPage("FEC Status & Counters", func(p pontop.Page) {
			emitInt(mw, Namespace+"_fec_bip_errors_total", p.KV["BIP errors"])
			emitInt(mw, Namespace+"_fec_codewords_total", p.KV["Total FEC codewords"], "outcome", "total")
			emitInt(mw, Namespace+"_fec_codewords_total", p.KV["Corrected FEC codewords"], "outcome", "corrected")
			emitInt(mw, Namespace+"_fec_codewords_total", p.KV["Uncorrected FEC codewords"], "outcome", "uncorrected")
			emitInt(mw, Namespace+"_fec_corrected_bytes_total", p.KV["Corrected FEC bytes"])
			emitInt(mw, Namespace+"_fec_errored_seconds_total", p.KV["FEC errored seconds"])
		})
	})
}

func collectGTC(mw *metrics.Writer) {
	mw.Header(Namespace+"_gtc_hec_errors_total", "GTC/XGTC HEC errors, by block and outcome.", metrics.Counter)
	mw.Header(Namespace+"_gtc_lost_words_total", "Words lost due to uncorrectable HEC errors.", metrics.Counter)
	mw.Header(Namespace+"_gtc_ploam_mic_errors_total", "PLOAM MIC integrity check errors.", metrics.Counter)

	scrape(mw, "pontop_gtc", func() error {
		return runPage("GTC/XGTC Status & Counters", func(p pontop.Page) {
			emitInt(mw, Namespace+"_gtc_hec_errors_total", p.KV["PSBd HEC errors corrected"], "block", "psbd", "outcome", "corrected")
			emitInt(mw, Namespace+"_gtc_hec_errors_total", p.KV["PSBd HEC errors uncorrected"], "block", "psbd", "outcome", "uncorrected")
			emitInt(mw, Namespace+"_gtc_hec_errors_total", p.KV["FS HEC errors corrected"], "block", "fs", "outcome", "corrected")
			emitInt(mw, Namespace+"_gtc_hec_errors_total", p.KV["FS HEC errors uncorrected"], "block", "fs", "outcome", "uncorrected")
			emitInt(mw, Namespace+"_gtc_lost_words_total", p.KV["Lost words due to HEC errors"])
			emitInt(mw, Namespace+"_gtc_ploam_mic_errors_total", p.KV["PLOAM MIC errors"])
		})
	})
}

func collectAlarms(mw *metrics.Writer) {
	mw.Header(Namespace+"_active_alarms", "Number of active alarms currently reported by the ONU.", metrics.Gauge)
	mw.Header(Namespace+"_active_alarm_info", "One sample per active alarm; value is always 1.", metrics.Gauge)

	scrape(mw, "pontop_alarms", func() error {
		return runPage("Active alarms", func(p pontop.Page) {
			mw.Sample(Namespace+"_active_alarms", float64(len(p.Rows)))
			for _, r := range p.Rows {
				mw.Sample(Namespace+"_active_alarm_info", 1,
					"type", r["Alarm type"],
					"alarm", r["Alarm"],
					"description", r["Description"],
				)
			}
		})
	})
}

func collectPLOAMMessages(mw *metrics.Writer) {
	mw.Header(Namespace+"_ploam_messages_total", "Cumulative PLOAM messages observed, by direction and type.", metrics.Counter)

	for _, dir := range []struct{ page, label string }{
		{"PLOAM Downstream Counters", "downstream"},
		{"PLOAM Upstream Counters", "upstream"},
	} {
		dir := dir
		scrape(mw, "pontop_ploam_"+dir.label, func() error {
			return runPage(dir.page, func(p pontop.Page) {
				for _, k := range p.Keys {
					n, err := pontop.FirstInt(p.KV[k])
					if err != nil {
						continue
					}
					mw.Sample(Namespace+"_ploam_messages_total", float64(n),
						"direction", dir.label,
						"type", snakeCase(k),
					)
				}
			})
		})
	}
}

func collectAllocations(mw *metrics.Writer) {
	mw.Header(Namespace+"_allocation_count_total", "Number of allocations granted to each alloc-id.", metrics.Counter)
	mw.Header(Namespace+"_allocation_idle_frames_total", "Idle frames sent against each alloc-id.", metrics.Counter)
	mw.Header(Namespace+"_allocation_upstream_bandwidth_bps", "Configured upstream bandwidth for each alloc-id (bps).", metrics.Gauge)
	mw.Header(Namespace+"_allocation_status_info", "Alloc status; value is always 1, useful labels code and description.", metrics.Gauge)

	scrape(mw, "pontop_allocations", func() error {
		return runPage("Allocation Counters", func(p pontop.Page) {
			for _, r := range p.Rows {
				id := r["Alloc id"]
				if id == "" {
					continue
				}
				if n, err := pontop.FirstInt(r["Allocations"]); err == nil {
					mw.Sample(Namespace+"_allocation_count_total", float64(n), "alloc_id", id)
				}
				if n, err := pontop.FirstInt(r["Alloc idle frames"]); err == nil {
					mw.Sample(Namespace+"_allocation_idle_frames_total", float64(n), "alloc_id", id)
				}
				if n, err := pontop.FirstInt(r["Upstream Bandwidth"]); err == nil {
					mw.Sample(Namespace+"_allocation_upstream_bandwidth_bps", float64(n), "alloc_id", id)
				}
				// Status is e.g. "3 LINKED". Split into code + description.
				code, desc := splitStatus(r["Status"])
				mw.Sample(Namespace+"_allocation_status_info", 1,
					"alloc_id", id, "code", code, "description", desc)
			}
		})
	})
}

func splitStatus(s string) (code, desc string) {
	s = strings.TrimSpace(s)
	parts := strings.SplitN(s, " ", 2)
	if len(parts) == 2 {
		return parts[0], strings.TrimSpace(parts[1])
	}
	return s, ""
}

// emitInt emits one sample whose value is parsed from the leading integer of
// raw. Silently skips on parse failure.
func emitInt(mw *metrics.Writer, name, raw string, labels ...string) {
	if n, err := pontop.FirstInt(raw); err == nil {
		mw.Sample(name, float64(n), labels...)
	}
}

// emitBool emits one sample iff raw is a recognised boolean keyword
// (ON/OFF, ENABLED/DISABLED, OK/ERROR, YES/NO, TRUE/FALSE).
func emitBool(mw *metrics.Writer, name, raw string, labels ...string) {
	if v, ok := pontop.Bool(raw); ok {
		mw.Sample(name, v, labels...)
	}
}
