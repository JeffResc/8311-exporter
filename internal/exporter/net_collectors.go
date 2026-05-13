package exporter

import (
	"strings"

	"github.com/jeffresc/8311-exporter/internal/metrics"
	"github.com/jeffresc/8311-exporter/internal/netstats"
)

// collectInterfaces enumerates /sys/class/net, then for each interface reads
// sysfs statistics and ethtool driver stats.
func collectInterfaces(mw *metrics.Writer, includeEthtool bool) {
	// Standard headers up front so output is well-formed even with zero
	// interfaces (shouldn't happen, but defensive).
	mw.Header(Namespace+"_interface_info", "Per-interface metadata; value is always 1.", metrics.Gauge)
	mw.Header(Namespace+"_interface_up", "1 if operstate is 'up', 0 otherwise.", metrics.Gauge)
	mw.Header(Namespace+"_interface_carrier", "1 if link carrier is present, 0 otherwise.", metrics.Gauge)
	mw.Header(Namespace+"_interface_speed_megabits", "Link speed reported by the driver, in Mb/s; absent when N/A.", metrics.Gauge)
	mw.Header(Namespace+"_interface_mtu_bytes", "Configured MTU, in bytes.", metrics.Gauge)
	mw.Header(Namespace+"_interface_statistic", "Per-interface kernel counter (from /sys/class/net/<if>/statistics/<name>).", metrics.Counter)
	if includeEthtool {
		mw.Header(Namespace+"_ethtool_statistic", "Per-interface driver-private counter (ethtool -S).", metrics.Counter)
	}

	scrape(mw, "interfaces", func() error {
		names, err := netstats.List()
		if err != nil {
			return err
		}
		for _, name := range names {
			iface, err := netstats.Read(name)
			if err != nil {
				continue
			}
			emitInterface(mw, iface, includeEthtool)
		}
		return nil
	})
}

func emitInterface(mw *metrics.Writer, iface netstats.Interface, includeEthtool bool) {
	mw.Sample(Namespace+"_interface_info", 1,
		"interface", iface.Name,
		"address", iface.Address,
		"operstate", iface.OperState,
	)
	mw.Sample(Namespace+"_interface_up", boolFloat(strings.EqualFold(iface.OperState, "up")), "interface", iface.Name)
	if iface.Carrier >= 0 {
		mw.Sample(Namespace+"_interface_carrier", float64(iface.Carrier), "interface", iface.Name)
	}
	if iface.SpeedMbps >= 0 {
		mw.Sample(Namespace+"_interface_speed_megabits", float64(iface.SpeedMbps), "interface", iface.Name)
	}
	if iface.MTU >= 0 {
		mw.Sample(Namespace+"_interface_mtu_bytes", float64(iface.MTU), "interface", iface.Name)
	}
	for stat, v := range iface.Statistics {
		mw.Sample(Namespace+"_interface_statistic", float64(v),
			"interface", iface.Name,
			"name", stat,
		)
	}
	if !includeEthtool {
		return
	}
	m, err := netstats.Ethtool(iface.Name)
	if err != nil || m == nil {
		return
	}
	for stat, v := range m {
		mw.Sample(Namespace+"_ethtool_statistic", float64(v),
			"interface", iface.Name,
			"name", stat,
		)
	}
}

func boolFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
