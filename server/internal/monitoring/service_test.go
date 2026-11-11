package monitoring

import (
	"testing"
	"time"
)

func TestMetricFreshness(t *testing.T) {
	now := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	staleAfter := 2 * time.Minute
	tests := []struct {
		name     string
		observed time.Time
		want     string
	}{
		{name: "missing", want: "missing"},
		{name: "fresh", observed: now.Add(-staleAfter), want: "fresh"},
		{name: "stale", observed: now.Add(-staleAfter - time.Nanosecond), want: "stale"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := metricFreshness(tt.observed, now, staleAfter); got != tt.want {
				t.Fatalf("metricFreshness() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHistoryObservedAtUsesLatestSampleTime(t *testing.T) {
	items := []Point{
		{SampledAt: time.Date(2026, 11, 12, 11, 0, 0, 0, time.UTC)},
		{SampledAt: time.Date(2026, 11, 12, 11, 5, 0, 0, time.UTC)},
	}
	want := items[1].SampledAt
	if got := historyObservedAt(items); !got.Equal(want) {
		t.Fatalf("historyObservedAt() = %s, want %s", got, want)
	}
}

func TestMissingWindowsAndLimit(t *testing.T) {
	start := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	items := []Point{{SampledAt: start}, {SampledAt: start.Add(10 * time.Minute)}}
	missing, truncated := missingWindows(items, start, start.Add(10*time.Minute), 5*time.Minute, 10)
	if truncated || len(missing) != 1 || !missing[0].Equal(start.Add(5*time.Minute)) {
		t.Fatalf("missing=%v truncated=%v", missing, truncated)
	}
	_, truncated = missingWindows(nil, start, start.Add(time.Hour), 5*time.Minute, 2)
	if !truncated {
		t.Fatal("window limit was not reported")
	}
}
