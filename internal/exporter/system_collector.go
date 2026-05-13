package exporter

import (
	"github.com/jeffresc/8311-exporter/internal/metrics"
	"github.com/jeffresc/8311-exporter/internal/system"
)

func collectSystem(mw *metrics.Writer) {
	mw.Header(Namespace+"_load1", "1-minute load average.", metrics.Gauge)
	mw.Header(Namespace+"_load5", "5-minute load average.", metrics.Gauge)
	mw.Header(Namespace+"_load15", "15-minute load average.", metrics.Gauge)
	mw.Header(Namespace+"_processes_runnable", "Currently runnable scheduling entities.", metrics.Gauge)
	mw.Header(Namespace+"_processes_total", "Total scheduling entities.", metrics.Gauge)
	mw.Header(Namespace+"_uptime_seconds", "Seconds since system boot.", metrics.Counter)
	mw.Header(Namespace+"_idle_seconds", "Cumulative idle CPU time since boot, summed across CPUs.", metrics.Counter)
	mw.Header(Namespace+"_memory_bytes", "Memory statistic from /proc/meminfo, by name.", metrics.Gauge)

	scrape(mw, "system_load", func() error {
		la, err := system.ReadLoadAvg()
		if err != nil {
			return err
		}
		mw.Sample(Namespace+"_load1", la.One)
		mw.Sample(Namespace+"_load5", la.Five)
		mw.Sample(Namespace+"_load15", la.Fifteen)
		mw.Sample(Namespace+"_processes_runnable", float64(la.Runnable))
		mw.Sample(Namespace+"_processes_total", float64(la.Total))
		return nil
	})

	scrape(mw, "system_uptime", func() error {
		u, err := system.ReadUptime()
		if err != nil {
			return err
		}
		mw.Sample(Namespace+"_uptime_seconds", u.UpSeconds)
		mw.Sample(Namespace+"_idle_seconds", u.IdleSeconds)
		return nil
	})

	scrape(mw, "system_memory", func() error {
		mi, err := system.ReadMemInfo()
		if err != nil {
			return err
		}
		for k, v := range mi {
			mw.Sample(Namespace+"_memory_bytes", float64(v), "name", k)
		}
		return nil
	})
}
