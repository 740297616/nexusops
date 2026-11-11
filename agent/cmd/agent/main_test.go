package main

import (
	"testing"

	"github.com/740297616/nexusops/agent/internal/metrics"
)

func TestMetricQueueRetainsNewestWhenFull(t *testing.T) {
	queue := make(chan metrics.Batch, 2)
	enqueueMetric(queue, metrics.Batch{RequestID: "first"})
	enqueueMetric(queue, metrics.Batch{RequestID: "second"})
	enqueueMetric(queue, metrics.Batch{RequestID: "third"})
	if (<-queue).RequestID != "second" || (<-queue).RequestID != "third" {
		t.Fatal("full queue did not discard the oldest batch")
	}
}
