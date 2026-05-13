package exporter

import "github.com/jeffresc/8311-exporter/internal/metrics"

// Version is set at link time via -ldflags '-X main.version=...' and mirrored
// here from main.go so the package boundary stays clean.
var Version = "dev"

func emitBuildInfo(mw *metrics.Writer) {
	mw.Header(Namespace+"_exporter_build_info", "8311-exporter build information; value is always 1.", metrics.Gauge)
	mw.Sample(Namespace+"_exporter_build_info", 1, "version", Version)
}
