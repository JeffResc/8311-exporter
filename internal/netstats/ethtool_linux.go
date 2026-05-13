// ethtool_linux.go implements per-interface stats fetch via the
// ETHTOOL_GSSET_INFO + ETHTOOL_GSTRINGS + ETHTOOL_GSTATS ioctl sequence.
//
// GSSET_INFO is required (rather than GDRVINFO) because several vendor
// netdevs on this firmware (PON GEM ports, packet-mappers) don't implement
// GDRVINFO but do implement the stats path. Buffers are sized to header
// + n * entry to keep per-call allocations in the low-KB range.

package netstats

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	siocEthtool = 0x8946

	ethSSStats    = 1
	ethGstringLen = 32

	ethtoolGstrings  = 0x0000001b
	ethtoolGstats    = 0x0000001d
	ethtoolGssetInfo = 0x00000037
)

// ifreq matches `struct ifreq` from <linux/if.h>: 16-byte name followed by a
// 16-byte union. SIOCETHTOOL only reads the data pointer at offset 16.
type ifreq struct {
	name [16]byte
	data uintptr
}

// ssetInfoCmd is `struct ethtool_sset_info` (fixed size — sset_mask asks for
// one set, so we only need a single trailing data slot).
type ssetInfoCmd struct {
	cmd      uint32
	reserved uint32
	ssetMask uint64
	count    uint32
}

// Ethtool reads driver-private counters for iface. Returns an empty map (not
// an error) when the driver advertises zero stats.
func Ethtool(iface string) (map[string]int64, error) {
	if len(iface) >= 16 {
		return nil, fmt.Errorf("ethtool: interface name too long: %q", iface)
	}
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("ethtool socket: %w", err)
	}
	defer func() { _ = unix.Close(fd) }()

	info := ssetInfoCmd{cmd: ethtoolGssetInfo, ssetMask: 1 << ethSSStats}
	if err := ethtoolCall(fd, iface, unsafe.Pointer(&info)); err != nil {
		return nil, fmt.Errorf("ethtool GSSET_INFO %s: %w", iface, err)
	}
	n := int(info.count)
	if n == 0 {
		return map[string]int64{}, nil
	}

	// ETHTOOL_GSTRINGS request: 12-byte header + n * 32-byte names.
	strBuf := make([]byte, 12+n*ethGstringLen)
	binary.NativeEndian.PutUint32(strBuf[0:4], ethtoolGstrings)
	binary.NativeEndian.PutUint32(strBuf[4:8], ethSSStats)
	binary.NativeEndian.PutUint32(strBuf[8:12], uint32(n))
	if err := ethtoolCall(fd, iface, unsafe.Pointer(&strBuf[0])); err != nil {
		return nil, fmt.Errorf("ethtool GSTRINGS %s: %w", iface, err)
	}

	// ETHTOOL_GSTATS request: 8-byte header + n * 8-byte u64 values.
	valBuf := make([]byte, 8+n*8)
	binary.NativeEndian.PutUint32(valBuf[0:4], ethtoolGstats)
	binary.NativeEndian.PutUint32(valBuf[4:8], uint32(n))
	if err := ethtoolCall(fd, iface, unsafe.Pointer(&valBuf[0])); err != nil {
		return nil, fmt.Errorf("ethtool GSTATS %s: %w", iface, err)
	}

	out := make(map[string]int64, n)
	for i := 0; i < n; i++ {
		nameBytes := strBuf[12+i*ethGstringLen : 12+(i+1)*ethGstringLen]
		if z := bytes.IndexByte(nameBytes, 0); z >= 0 {
			nameBytes = nameBytes[:z]
		}
		if len(nameBytes) == 0 {
			continue
		}
		v := binary.NativeEndian.Uint64(valBuf[8+i*8 : 16+i*8])
		out[string(nameBytes)] = int64(v)
	}
	return out, nil
}

func ethtoolCall(fd int, iface string, data unsafe.Pointer) error {
	var req ifreq
	copy(req.name[:], iface)
	req.data = uintptr(data)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), siocEthtool, uintptr(unsafe.Pointer(&req)))
	if errno != 0 {
		return errno
	}
	return nil
}
