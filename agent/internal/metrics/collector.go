package metrics

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

type Collector struct {
	mu       sync.Mutex
	previous map[string]NetworkCounter
}

func NewCollector() *Collector { return &Collector{previous: map[string]NetworkCounter{}} }

func (c *Collector) Collect(ctx context.Context) (Batch, error) {
	now := time.Now().UTC()
	batch := Batch{ProtocolVersion: "1", SchemaVersion: 1, SampledAt: now}
	var errs []string
	add := func(metricType, name string, labels map[string]string, value float64, unit string) {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			return
		}
		batch.Samples = append(batch.Samples, Sample{MetricType: metricType, MetricName: name, Labels: labels, Value: value, Unit: unit})
	}
	if values, err := cpu.PercentWithContext(ctx, 0, false); err == nil && len(values) > 0 {
		add("cpu", "usage_percent", nil, values[0], "percent")
	} else if err != nil {
		errs = append(errs, "cpu usage")
	}
	if cores, err := cpu.CountsWithContext(ctx, true); err == nil {
		add("cpu", "cores", nil, float64(cores), "count")
	} else {
		errs = append(errs, "cpu cores")
	}
	if avg, err := load.AvgWithContext(ctx); err == nil {
		add("cpu", "load_1m", nil, avg.Load1, "load")
		add("cpu", "load_5m", nil, avg.Load5, "load")
		add("cpu", "load_15m", nil, avg.Load15, "load")
	} else {
		errs = append(errs, "load")
	}
	if value, err := mem.VirtualMemoryWithContext(ctx); err == nil {
		add("memory", "total_bytes", nil, float64(value.Total), "bytes")
		add("memory", "used_bytes", nil, float64(value.Used), "bytes")
		add("memory", "available_bytes", nil, float64(value.Available), "bytes")
		add("memory", "usage_percent", nil, value.UsedPercent, "percent")
	} else {
		errs = append(errs, "memory")
	}
	if partitions, err := disk.PartitionsWithContext(ctx, false); err == nil {
		for _, part := range partitions {
			usage, usageErr := disk.UsageWithContext(ctx, part.Mountpoint)
			if usageErr != nil {
				continue
			}
			add("filesystem", "total_bytes", map[string]string{"mountpoint": part.Mountpoint, "filesystem": part.Fstype}, float64(usage.Total), "bytes")
			add("filesystem", "used_bytes", map[string]string{"mountpoint": part.Mountpoint, "filesystem": part.Fstype}, float64(usage.Used), "bytes")
			add("filesystem", "available_bytes", map[string]string{"mountpoint": part.Mountpoint, "filesystem": part.Fstype}, float64(usage.Free), "bytes")
			add("filesystem", "usage_percent", map[string]string{"mountpoint": part.Mountpoint, "filesystem": part.Fstype}, usage.UsedPercent, "percent")
		}
	} else {
		errs = append(errs, "filesystem")
	}
	if counters, err := net.IOCountersWithContext(ctx, true); err == nil {
		c.mu.Lock()
		defer c.mu.Unlock()
		for _, value := range counters {
			key := value.Name
			current := NetworkCounter{RXBytes: value.BytesRecv, TXBytes: value.BytesSent, At: now}
			labels := map[string]string{"interface": key}
			add("network", "rx_bytes_total", labels, float64(value.BytesRecv), "bytes")
			add("network", "tx_bytes_total", labels, float64(value.BytesSent), "bytes")
			if previous, ok := c.previous[key]; ok {
				rx, tx, valid := Rate(previous, current)
				if valid {
					add("network", "rx_bytes_per_second", labels, rx, "bytes_per_second")
					add("network", "tx_bytes_per_second", labels, tx, "bytes_per_second")
				}
			}
			c.previous[key] = current
		}
	} else {
		errs = append(errs, "network")
	}
	if len(batch.Samples) == 0 {
		return batch, fmt.Errorf("no metrics collected: %s", strings.Join(errs, ", "))
	}
	return batch, nil
}
