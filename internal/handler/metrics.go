// Package handler implements the HTTP handlers and router for the metrics
// collection server: updating and reading gauge/counter metrics, the HTML
// index page, and the database health check.
package handler

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/audit"
	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/repository"
	"go.uber.org/zap"
)

// MetricsHandler serves the metric update/read endpoints. It persists
// metrics through a repository.Storage and, on every successful update,
// notifies an audit.Auditor about the request.
type MetricsHandler struct {
	storage repository.Storage
	log     *zap.Logger
	auditor *audit.Auditor
}

// NewMetricsHandler creates a MetricsHandler backed by the given storage.
// auditor may have no observers attached, in which case audit notifications
// are silently dropped.
func NewMetricsHandler(s repository.Storage, log *zap.Logger, auditor *audit.Auditor) *MetricsHandler {
	return &MetricsHandler{storage: s, log: log, auditor: auditor}
}

// Update handles POST /update/{type}/{name}/{value}: it parses a single
// gauge or counter value from the URL path and stores it. It responds with
// 400 for an unknown type or an unparsable value, and 200 on success.
func (h *MetricsHandler) Update(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	metricsType := chi.URLParam(r, "type")
	metricsName := chi.URLParam(r, "name")
	metricsValue := chi.URLParam(r, "value")

	if metricsName == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	switch metricsType {
	case models.Gauge:
		value, err := strconv.ParseFloat(metricsValue, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := h.storage.UpdateGauge(metricsName, value); err != nil {
			h.log.Error("failed to update gauge", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	case models.Counter:
		value, err := strconv.ParseInt(metricsValue, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := h.storage.UpdateCounter(metricsName, value); err != nil {
			h.log.Error("failed to update counter", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	h.auditor.Notify(audit.Event{
		IPAddress: host,
		Metrics:   []string{metricsName},
		Timestamp: time.Now().Unix(),
	})
	w.WriteHeader(http.StatusOK)
}

// Value handles GET /value/{type}/{name}: it returns the current value of a
// gauge or counter as plain text. It responds with 404 if the metric type is
// unknown or the metric was never recorded.
func (h *MetricsHandler) Value(w http.ResponseWriter, r *http.Request) {
	metricsType := chi.URLParam(r, "type")
	metricsName := chi.URLParam(r, "name")

	switch metricsType {
	case models.Gauge:
		value, err := h.storage.Gauge(metricsName)
		if errors.Is(err, repository.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err != nil {
			h.log.Error("failed to read gauge", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if _, err = w.Write([]byte(strconv.FormatFloat(value, 'f', -1, 64))); err != nil {
			h.log.Error("failed to write response", zap.Error(err))
		}
	case models.Counter:
		value, err := h.storage.Counter(metricsName)
		if errors.Is(err, repository.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err != nil {
			h.log.Error("failed to read counter", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if _, err = w.Write([]byte(strconv.FormatInt(value, 10))); err != nil {
			h.log.Error("failed to write response", zap.Error(err))
		}
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// Index handles GET /: it renders an HTML page listing every known gauge and
// counter and their current values.
func (h *MetricsHandler) Index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	var b strings.Builder

	for name, value := range h.storage.AllGauges() {
		b.WriteString("<p>")
		b.WriteString(name)
		b.WriteString(": ")
		b.WriteString(strconv.FormatFloat(value, 'f', -1, 64))
		b.WriteString("</p>")
	}

	for name, value := range h.storage.AllCounters() {
		b.WriteString("<p>")
		b.WriteString(name)
		b.WriteString(": ")
		b.WriteString(strconv.FormatInt(value, 10))
		b.WriteString("</p>")
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(b.String())); err != nil {
		h.log.Error("failed to write response", zap.Error(err))
	}
}

// UpdateJSON handles POST /update: it decodes a single models.Metrics value
// from the request body, stores it, and echoes it back as JSON (with Delta
// set to the new cumulative total for counters). It responds with 400 for
// malformed input or an unknown/missing value.
func (h *MetricsHandler) UpdateJSON(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	model := models.Metrics{}

	err := json.NewDecoder(r.Body).Decode(&model)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch model.MType {
	case models.Gauge:
		if model.Value == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err = h.storage.UpdateGauge(model.ID, *model.Value); err != nil {
			h.log.Error("failed to update gauge", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	case models.Counter:
		if model.Delta == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err = h.storage.UpdateCounter(model.ID, *model.Delta); err != nil {
			h.log.Error("failed to update counter", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var total int64
		if total, err = h.storage.Counter(model.ID); err == nil {
			model.Delta = &total
		}
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	data, err := json.Marshal(model)
	if err != nil {
		h.log.Error("failed to marshal metric", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	h.auditor.Notify(audit.Event{
		IPAddress: host,
		Metrics:   []string{model.ID},
		Timestamp: time.Now().Unix(),
	})
	w.WriteHeader(http.StatusOK)
	if _, err = w.Write(data); err != nil {
		h.log.Error("failed to write response", zap.Error(err))
	}
}

// ValueJSON handles POST /value: it decodes a models.Metrics value carrying
// only ID and MType from the request body and responds with the full metric
// (Value or Delta populated) as JSON. It responds with 404 if the metric was
// never recorded.
func (h *MetricsHandler) ValueJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	model := models.Metrics{}

	err := json.NewDecoder(r.Body).Decode(&model)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch model.MType {
	case models.Gauge:
		var value float64
		value, err = h.storage.Gauge(model.ID)
		if errors.Is(err, repository.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err != nil {
			h.log.Error("failed to read gauge", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		model.Value = &value
	case models.Counter:
		var value int64
		value, err = h.storage.Counter(model.ID)
		if errors.Is(err, repository.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err != nil {
			h.log.Error("failed to read counter", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		model.Delta = &value
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}

	data, err := json.Marshal(model)
	if err != nil {
		h.log.Error("failed to marshal metric", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	if _, err = w.Write(data); err != nil {
		h.log.Error("failed to write response", zap.Error(err))
	}
}

// UpdateBatch handles POST /updates: it decodes a JSON array of
// models.Metrics from the request body and stores all of them in a single
// call to the underlying storage. An empty array is accepted and is a no-op.
func (h *MetricsHandler) UpdateBatch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var metrics []models.Metrics

	err := json.NewDecoder(r.Body).Decode(&metrics)
	if err != nil {
		h.log.Error("failed to unmarshal metrics", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(metrics) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	err = h.storage.UpdateBatch(metrics)
	if err != nil {
		h.log.Error("failed to update metrics", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	slice := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		slice = append(slice, metric.ID)
	}
	h.auditor.Notify(audit.Event{
		IPAddress: host,
		Metrics:   slice,
		Timestamp: time.Now().Unix(),
	})
	w.WriteHeader(http.StatusOK)

}
