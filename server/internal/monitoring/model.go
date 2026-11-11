package monitoring

import "time"

type Sample struct {
	MetricType string            `json:"metric_type"`
	MetricName string            `json:"metric_name"`
	Labels     map[string]string `json:"labels"`
	Value      float64           `json:"value"`
	Unit       string            `json:"unit"`
}

type IngestRequest struct {
	ProtocolVersion string    `json:"protocol_version"`
	SchemaVersion   int       `json:"schema_version"`
	RequestID       string    `json:"request_id"`
	SampledAt       time.Time `json:"sampled_at"`
	Samples         []Sample  `json:"samples"`
}

type IngestResult struct {
	Accepted   int       `json:"accepted"`
	ReceivedAt time.Time `json:"received_at"`
	RequestID  string    `json:"request_id"`
}

type Point struct {
	MetricType string            `json:"metric_type"`
	MetricName string            `json:"metric_name"`
	Labels     map[string]string `json:"labels"`
	Value      float64           `json:"value"`
	Unit       string            `json:"unit"`
	SampledAt  time.Time         `json:"sampled_at"`
	ReceivedAt time.Time         `json:"received_at"`
}

type Latest struct {
	ServerID   string    `json:"server_id"`
	Status     string    `json:"status"`
	ObservedAt time.Time `json:"observed_at"`
	Stale      bool      `json:"stale"`
	Metrics    []Point   `json:"metrics"`
}

type History struct {
	ServerID    string  `json:"server_id"`
	MetricType  string  `json:"metric_type"`
	MetricName  string  `json:"metric_name"`
	Aggregation string  `json:"aggregation"`
	Stale       bool    `json:"stale"`
	Series      []Point `json:"series"`
}
