package metrics

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

func fakeCollector() *Collector {
	c := NewCollector()
	c.providers = providers{
		cpuPercent: func(context.Context, time.Duration, bool) ([]float64, error) { return []float64{42}, nil },
		cpuCounts:  func(context.Context, bool) (int, error) { return 4, nil },
		load:       func(context.Context) (*load.AvgStat, error) { return &load.AvgStat{Load1: 1}, nil },
		memory: func(context.Context) (*mem.VirtualMemoryStat, error) {
			return &mem.VirtualMemoryStat{Total: 100, Used: 40, Available: 60, UsedPercent: 40}, nil
		},
		partitions: func(context.Context, bool) ([]disk.PartitionStat, error) {
			return []disk.PartitionStat{{Mountpoint: "/", Fstype: "ext4"}, {Mountpoint: "/data", Fstype: "xfs"}}, nil
		},
		diskUsage: func(context.Context, string) (*disk.UsageStat, error) {
			return &disk.UsageStat{Total: 100, Used: 30, Free: 70, UsedPercent: 30}, nil
		},
		network: func(context.Context, bool) ([]net.IOCountersStat, error) {
			return []net.IOCountersStat{{Name: "eth0", BytesRecv: 100, BytesSent: 200}, {Name: "eth1", BytesRecv: 200, BytesSent: 300}}, nil
		},
	}
	return c
}

func countMetric(batch Batch, typ, name string) int {
	count := 0
	for _, sample := range batch.Samples {
		if sample.MetricType == typ && sample.MetricName == name {
			count++
		}
	}
	return count
}

func TestCollectorIndependentSources(t *testing.T) {
	c := fakeCollector()
	batch, err := c.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		typ, name string
		want      int
	}{{"cpu", "usage_percent", 1}, {"memory", "usage_percent", 1}, {"filesystem", "usage_percent", 2}, {"network", "rx_bytes_total", 2}, {"network", "rx_bytes_per_second", 0}} {
		if got := countMetric(batch, test.typ, test.name); got != test.want {
			t.Errorf("%s/%s count = %d, want %d", test.typ, test.name, got, test.want)
		}
	}

	failure := errors.New("collector unavailable")
	c.providers.cpuPercent = func(context.Context, time.Duration, bool) ([]float64, error) { return nil, failure }
	c.providers.memory = func(context.Context) (*mem.VirtualMemoryStat, error) { return nil, failure }
	c.providers.partitions = func(context.Context, bool) ([]disk.PartitionStat, error) { return nil, failure }
	batch, err = c.Collect(context.Background())
	if err != nil || countMetric(batch, "cpu", "usage_percent") != 0 || countMetric(batch, "memory", "usage_percent") != 0 || countMetric(batch, "filesystem", "usage_percent") != 0 || countMetric(batch, "network", "rx_bytes_total") != 2 {
		t.Fatalf("partial collection failed: samples=%v err=%v", batch.Samples, err)
	}
}

func TestCollectorNetworkResetAndInterfaceChanges(t *testing.T) {
	c := fakeCollector()
	if _, err := c.Collect(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Make the prior samples old enough for a deterministic positive interval.
	for name, counter := range c.previous {
		counter.At = counter.At.Add(-time.Second)
		c.previous[name] = counter
	}
	c.providers.network = func(context.Context, bool) ([]net.IOCountersStat, error) {
		return []net.IOCountersStat{{Name: "eth0", BytesRecv: 50, BytesSent: 50}, {Name: "eth2", BytesRecv: 100, BytesSent: 100}}, nil
	}
	batch, err := c.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if countMetric(batch, "network", "rx_bytes_per_second") != 0 {
		t.Fatal("reset or new interface produced a rate")
	}
	if _, ok := c.previous["eth1"]; ok {
		t.Fatal("removed interface retained its baseline")
	}
	c.providers.network = func(context.Context, bool) ([]net.IOCountersStat, error) {
		return []net.IOCountersStat{{Name: "eth1", BytesRecv: 300, BytesSent: 400}}, nil
	}
	batch, err = c.Collect(context.Background())
	if err != nil || countMetric(batch, "network", "rx_bytes_per_second") != 0 {
		t.Fatal("reappearing interface must establish a fresh baseline")
	}
}
