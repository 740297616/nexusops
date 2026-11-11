package metrics

import (
	"context"
	"testing"
	"time"
)

func TestRateFirstAndCounterReset(t *testing.T) {
	first := NetworkCounter{RXBytes: 100, TXBytes: 200, At: time.Unix(0, 0)}
	second := NetworkCounter{RXBytes: 300, TXBytes: 600, At: time.Unix(2, 0)}
	rx, tx, valid := Rate(first, second)
	if !valid || rx != 100 || tx != 200 {
		t.Fatalf("rate = %v, %v, %v", rx, tx, valid)
	}
	if _, _, valid := Rate(NetworkCounter{}, first); valid {
		t.Fatal("first sample must not produce a rate")
	}
	reset := NetworkCounter{RXBytes: 50, TXBytes: 60, At: time.Unix(3, 0)}
	if _, _, valid := Rate(second, reset); valid {
		t.Fatal("counter reset must invalidate rate")
	}
}

func TestCollectorReturnsSamples(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	batch, err := NewCollector().Collect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Samples) == 0 {
		t.Fatal("collector returned no samples")
	}
	if batch.SampledAt.IsZero() {
		t.Fatal("sample timestamp missing")
	}
}
