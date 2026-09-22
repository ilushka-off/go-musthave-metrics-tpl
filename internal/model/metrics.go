// Package models defines the shared metric data types exchanged between the
// agent, the server's HTTP API, and its storage layer.
package models

// Metric type discriminators used in Metrics.MType.
const (
	Counter = "counter"
	Gauge   = "gauge"
)

// NOTE: Не усложняем пример, вводя иерархическую вложенность структур.
// Органичиваясь плоской моделью.
// Delta и Value объявлены через указатели,
// что бы отличать значение "0", от не заданного значения
// и соответственно не кодировать в структуру.

// Metrics is a single metric, in the wire format used by the HTTP JSON API.
// Exactly one of Delta (for Counter) or Value (for Gauge) is populated,
// matching MType.
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}
