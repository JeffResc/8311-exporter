// Package optics parses SFF-8472 DDM (Digital Diagnostic Monitoring) data
// from the 8311 firmware's pon_mbox eeprom51 sysfs file.
//
// The math here mirrors /usr/lib/lua/8311/tools.lua::metrics() exactly so the
// exporter's values match the firmware's existing /cgi-bin/luci/8311/metrics
// endpoint byte-for-byte.
package optics

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
)

// EEPROM51Path is the firmware-exposed sysfs file containing the SFF-8472
// A2h diagnostic page bytes.
const EEPROM51Path = "/sys/class/pon_mbox/pon_mbox0/device/eeprom51"

// MinBytes is the smallest eeprom51 dump that contains every DDM field we read.
// SFF-8472 page A2h, bytes 96..105 inclusive (10 bytes), so 106 minimum.
const MinBytes = 106

// Readings is the decoded set of SFF-8472 DDM measurements.
type Readings struct {
	TempC      float64
	VoltageV   float64
	TxBiasMA   float64
	TxPowerDBm float64
	RxPowerDBm float64
	TxPowerMW  float64
	RxPowerMW  float64
}

// Read returns the decoded DDM readings from the firmware's eeprom51 file.
func Read() (Readings, error) {
	b, err := os.ReadFile(EEPROM51Path)
	if err != nil {
		return Readings{}, fmt.Errorf("read eeprom51: %w", err)
	}
	return Parse(b)
}

// Parse decodes the SFF-8472 A2h DDM fields starting at offset 96.
// Layout (offset : meaning : units):
//
//	 96 : temp MSB    : signed °C
//	 97 : temp LSB    : 1/256 °C (unsigned fractional, per Lua impl)
//	 98 : Vcc MSB     : 0.1 mV units, big-endian uint16 -> /10000 = V
//	100 : TX bias MSB : 2 µA units, big-endian uint16 -> /500 = mA
//	102 : TX power MSB: 0.1 µW units, big-endian uint16 -> /10000 = mW
//	104 : RX power MSB: 0.1 µW units, big-endian uint16 -> /10000 = mW
//
// The Lua reference uses unsigned bytes for temperature; we match that.
func Parse(b []byte) (Readings, error) {
	if len(b) < MinBytes {
		return Readings{}, fmt.Errorf("eeprom51 too short: %d bytes (need %d)", len(b), MinBytes)
	}

	temp := float64(b[96]) + float64(b[97])/256.0
	voltage := float64(binary.BigEndian.Uint16(b[98:100])) / 10000.0
	txBias := float64(binary.BigEndian.Uint16(b[100:102])) / 500.0
	txMW := float64(binary.BigEndian.Uint16(b[102:104])) / 10000.0
	rxMW := float64(binary.BigEndian.Uint16(b[104:106])) / 10000.0

	return Readings{
		TempC:      temp,
		VoltageV:   voltage,
		TxBiasMA:   txBias,
		TxPowerMW:  txMW,
		RxPowerMW:  rxMW,
		TxPowerDBm: mwToDBm(txMW),
		RxPowerDBm: mwToDBm(rxMW),
	}, nil
}

func mwToDBm(mw float64) float64 {
	if mw <= 0 {
		return math.Inf(-1)
	}
	return 10 * math.Log10(mw)
}
