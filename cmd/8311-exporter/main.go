// 8311-exporter is a Prometheus exporter for the 8311 community firmware
// (X-ONU-SFPP, WAS-110, and other XGS-PON SFP+ ONT sticks).
//
// It runs on-device, reads native data sources (sysfs, `pon`, `pontop`,
// ubus, ethtool), and exposes everything on /metrics. Metrics are prefixed
// `ont_8311_`.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/jeffresc/8311-exporter/internal/exporter"
)

// Set via -ldflags '-X main.version=...' at build time. Mirrored into the
// exporter package so the build_info gauge can include it.
var version = "dev"

func init() { exporter.Version = version }

func main() {
	addr := flag.String("listen", ":9833", "address to listen on")
	ethtool := flag.Bool("ethtool", true, "include per-interface ethtool driver stats (adds ~100ms per scrape via ioctl)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	// Keep memory footprint small on the 1 GB SFP stick: aggressive GC and a
	// soft 32 MB heap ceiling. Working set is ~9 MB so this leaves plenty of
	// headroom while preventing runaway growth under burst load.
	debug.SetGCPercent(50)
	debug.SetMemoryLimit(32 << 20)

	if *showVersion {
		fmt.Println("8311-exporter", version)
		return
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", exporter.Handler(*ethtool))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprintln(w, "8311-exporter — Prometheus metrics at /metrics")
	})

	srv := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("8311-exporter %s listening on %s", version, *addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Println("server:", err)
		os.Exit(1)
	}
}
