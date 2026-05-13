// Package thermal reads CPU temperatures from /sys/class/thermal.
package thermal

import (
	"fmt"

	"github.com/jeffresc/8311-exporter/internal/sysfs"
)

// Zone reads /sys/class/thermal/thermal_zone<n>/temp and returns °C.
// The kernel reports milli-degrees Celsius; we divide by 1000.
func Zone(n int) (float64, error) {
	return ZoneAt(fmt.Sprintf("/sys/class/thermal/thermal_zone%d/temp", n))
}

// ZoneAt reads a thermal_zone temperature file at the given path.
func ZoneAt(path string) (float64, error) {
	v, err := sysfs.ReadInt64(path)
	if err != nil {
		return 0, err
	}
	return float64(v) / 1000.0, nil
}
