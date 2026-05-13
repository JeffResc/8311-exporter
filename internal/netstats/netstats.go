// Package netstats reads per-interface kernel statistics from
// /sys/class/net/<iface>/{statistics/*,operstate,carrier,speed,mtu,address}
// and reads driver-private counters via the ETHTOOL_GSTATS ioctl (see
// ethtool_linux.go). Linux-only.
package netstats

import (
	"os"
	"path/filepath"

	"github.com/jeffresc/8311-exporter/internal/sysfs"
)

// SysClassNet is the root of the netdev sysfs hierarchy.
const SysClassNet = "/sys/class/net"

// Interface returns the metadata + statistics for one netdev.
type Interface struct {
	Name       string
	OperState  string           // "up", "down", "unknown"
	Carrier    int64            // 0 or 1; -1 if unreadable
	SpeedMbps  int64            // -1 if N/A
	MTU        int64            // -1 if unreadable
	Address    string           // MAC
	Statistics map[string]int64 // /statistics/* contents
}

// List enumerates the interfaces under /sys/class/net.
func List() ([]string, error) {
	entries, err := os.ReadDir(SysClassNet)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}

// Read returns the metadata + statistics for a single named interface.
// Missing files are skipped (not all interfaces expose every attribute).
func Read(name string) (Interface, error) {
	iface := Interface{
		Name:       name,
		Carrier:    -1,
		SpeedMbps:  -1,
		MTU:        -1,
		Statistics: map[string]int64{},
	}
	base := filepath.Join(SysClassNet, name)

	iface.OperState = sysfs.ReadString(filepath.Join(base, "operstate"))
	if v, err := sysfs.ReadInt64(filepath.Join(base, "carrier")); err == nil {
		iface.Carrier = v
	}
	if v, err := sysfs.ReadInt64(filepath.Join(base, "speed")); err == nil {
		iface.SpeedMbps = v
	}
	if v, err := sysfs.ReadInt64(filepath.Join(base, "mtu")); err == nil {
		iface.MTU = v
	}
	iface.Address = sysfs.ReadString(filepath.Join(base, "address"))

	statsDir := filepath.Join(base, "statistics")
	entries, err := os.ReadDir(statsDir)
	if err != nil {
		return iface, nil
	}
	for _, e := range entries {
		if v, err := sysfs.ReadInt64(filepath.Join(statsDir, e.Name())); err == nil {
			iface.Statistics[e.Name()] = v
		}
	}
	return iface, nil
}
