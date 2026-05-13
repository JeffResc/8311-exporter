# 8311-exporter

Prometheus exporter for the [8311 community firmware](https://github.com/djGrrr/8311-was-110-firmware-builder).
Runs natively on XGS-PON SFP+ ONT sticks (X-ONU-SFPP, WAS-110, and other
sticks flashed with the 8311 firmware) and exposes ~1300+ metrics covering
optics, PLOAM, FEC, GTC, per-GEM port counters, allocations, alarms, every
network interface plus its ethtool driver counters, and host metrics.

All metrics are prefixed `ont_8311_`.

## Quick start

```
task install HOST=root@192.168.98.1   # scp + restart, no IPK
curl http://192.168.98.1:9833/metrics
```

Or build and install the IPK (requires Docker for the OpenWrt SDK):

```
task install:ipk HOST=root@192.168.98.1 VERSION=0.5.1
```

## Resource usage

Measured on a 400 MHz PRX126 stick (21 netdevs, 11 with ethtool stats):

- Binary: ~6.4 MB stripped, statically linked (no glibc dependency).
- RSS: ~9 MB resident in steady state.
- Scrape: ~2.1 s end-to-end. Ethtool stats are read via the
  `ETHTOOL_GSTATS` ioctl.

## Configuration

UCI config at `/etc/config/8311-exporter`:

```
config 8311-exporter 'main'
    option enabled '1'
    option listen ':9833'
    option ethtool '1'
```

- `listen`: address:port to bind. Restrict to `127.0.0.1:9833` if you tunnel
  scrapes over WireGuard/etc.
- `ethtool`: set to `0` to skip per-interface driver counters (drops the
  metric count from ~1460 to ~970).

## Tasks

This project uses [Task](https://taskfile.dev/) instead of Make. All build,
test, and deploy flows are in [Taskfile.yml](Taskfile.yml).

```
task                  # default: test + cross-compile
task test             # go test ./...
task build            # host binary (for quick iteration)
task build:mips       # cross-compile linux/mips/softfloat
task ipk              # build .ipk via OpenWrt SDK in Docker
task install          # scp binary + procd restart
task install:ipk      # opkg install on the device
task clean            # remove bin/ and dist/
```

Pass `VERSION=...` and/or `HOST=root@<ip>` to override defaults. The
`ARCH=` var defaults to `mips_24kc_nomips16` (what the 8311 firmware's
opkg expects); set to `mips_24kc` if building for stock OpenWrt.

## Installing on a stick

Three options, in order of persistence.

### 1. `opkg install` (recommended)

```
task ipk VERSION=0.5.1
scp -O dist/8311-exporter_0.5.1-1_mips_24kc_nomips16.ipk root@<stick>:/tmp/
ssh root@<stick> opkg install /tmp/8311-exporter_0.5.1-1_mips_24kc_nomips16.ipk
```

The IPK ships the binary, the procd init script, and the uci config.
Survives reboots; does NOT survive flashing new firmware.

### 2. Bake into the firmware

Copy `packaging/openwrt/files/` and the cross-compiled binary into
[djGrrr's firmware-builder](https://github.com/djGrrr/8311-was-110-firmware-builder)
`mods/` overlay, then build and flash:

```
task build:mips VERSION=0.5.1
cp -a packaging/openwrt/files/* /path/to/8311-was-110-firmware-builder/mods/
cp bin/8311-exporter.mips        /path/to/8311-was-110-firmware-builder/mods/usr/sbin/8311-exporter
```

Survives firmware upgrades.

### 3. Plain scp (dev / try-before-you-buy)

```
task install HOST=root@<stick>
```

Just scps the binary and bounces the service. Convenient for iteration;
doesn't persist anywhere.

## Metrics inventory

Run the exporter and grep `# HELP` for the live inventory. High-level
families:

| Family | What |
|---|---|
| `ont_8311_optic_*` | SFF-8472 DDM: temperature, voltage, Tx bias, Tx/Rx power (dBm + mW) |
| `ont_8311_cpu_temperature_celsius` | Per-CPU thermal zones |
| `ont_8311_ploam_*` | PLOAM state, time-in-state, errorcode, state info |
| `ont_8311_ploam_messages_total` | Per-message-type PLOAM counters, up + down |
| `ont_8311_fec_*` | Codewords (total/corrected/uncorrected), corrected bytes, errored seconds |
| `ont_8311_gtc_*` | PSBd/FS HEC errors, lost words, PLOAM MIC errors |
| `ont_8311_active_alarms*` | Alarm count + per-alarm info |
| `ont_8311_gem_*` | Per-GEM-port packets, bytes, key errors, size histograms, status |
| `ont_8311_allocation_*` | Per-alloc-id allocations, idle frames, upstream bandwidth, status |
| `ont_8311_power_save_*` / `ont_8311_psm_*` | Power-saving state and counters |
| `ont_8311_capabilit*` / `ont_8311_optic_info` | Capability flags, vendor info (info-style gauges) |
| `ont_8311_interface_*` | Per-netdev sysfs counters, MTU, speed, carrier, operstate |
| `ont_8311_ethtool_statistic` | All `ethtool -S` driver-private counters, per interface |
| `ont_8311_load*` / `_uptime_seconds` / `_memory_bytes` | Host load, uptime, memory |
| `ont_8311_scrape_duration_seconds` / `_scrape_success` | Per-source exporter self-metrics |
| `ont_8311_exporter_build_info` | Exporter version |

## Development

Unit tests use captured fixtures under `testdata/` so they run anywhere —
no live device needed.

```
task test
```
