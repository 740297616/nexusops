package metrics

import "time"

type Sample struct {
	MetricType string            `json:"metric_type"`
	MetricName string            `json:"metric_name"`
	Labels     map[string]string `json:"labels"`
	Value      float64           `json:"value"`
	Unit       string            `json:"unit"`
}

type Batch struct {
	ProtocolVersion string    `json:"protocol_version"`
	SchemaVersion   int       `json:"schema_version"`
	RequestID       string    `json:"request_id"`
	SampledAt       time.Time `json:"sampled_at"`
	Samples         []Sample  `json:"samples"`
}

type NetworkCounter struct {
	RXBytes uint64
	TXBytes uint64
	At      time.Time
}

func Rate(previous, current NetworkCounter) (rx, tx float64, valid bool) {
	if previous.At.IsZero() || current.At.IsZero() {
		return 0, 0, false
	}
	delta := current.At.Sub(previous.At).Seconds()
	if delta <= 0 || current.RXBytes < previous.RXBytes || current.TXBytes < previous.TXBytes {
		return 0, 0, false
	}
	return float64(current.RXBytes-previous.RXBytes) / delta, float64(current.TXBytes-previous.TXBytes) / delta, true
}
