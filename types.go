package mget

import "errors"

// MetricType enum for metric type
type MetricType string

const (
	CounterType   MetricType = "COUNTER"
	GaugeType     MetricType = "GAUGE"
	HistogramType MetricType = "HISTOGRAM"
	SummaryType   MetricType = "SUMMARY"
	UnknownType   MetricType = "UNKNOWN"
)

var (
	ErrEndpointMissing = errors.New("missing endpoint specification")
)

// LabelValuePair defines a pair of labels and value
type LabelValuePair struct {
	Label map[string]string
	Value float64
}

// Metric defines a metric with all its label value pairs
type Metric struct {
	Name    string
	Help    string
	Type    MetricType
	TypeRaw string
	Values  []LabelValuePair
}

// Result holds the result of a request
type Result struct {
	ResponseStatus int
	Err            error
	Metric         []Metric
}
