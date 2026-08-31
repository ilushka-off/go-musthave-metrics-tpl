package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/repository"
	"go.uber.org/zap"
)

type MetricsHandler struct {
	storage repository.Storage
	log     *zap.Logger
}

func NewMetricsHandler(s repository.Storage, log *zap.Logger) *MetricsHandler {
	return &MetricsHandler{storage: s, log: log}
}

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
	w.WriteHeader(http.StatusOK)
}

func (h *MetricsHandler) Value(w http.ResponseWriter, r *http.Request) {
	metricsType := chi.URLParam(r, "type")
	metricsName := chi.URLParam(r, "name")

	switch metricsType {
	case models.Gauge:
		value, ok := h.storage.Gauge(metricsName)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(strconv.FormatFloat(value, 'f', -1, 64)))
	case models.Counter:
		value, ok := h.storage.Counter(metricsName)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(strconv.FormatInt(value, 10)))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (h *MetricsHandler) Index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	var b strings.Builder

	for name, value := range h.storage.AllGauges() {
		b.WriteString(fmt.Sprintf("<p>%s: %f</p>", name, value))
	}

	for name, value := range h.storage.AllCounters() {
		b.WriteString(fmt.Sprintf("<p>%s: %d</p>", name, value))
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(b.String()))
}

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
		if err := h.storage.UpdateGauge(model.ID, *model.Value); err != nil {
			h.log.Error("failed to update gauge", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	case models.Counter:
		if model.Delta == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := h.storage.UpdateCounter(model.ID, *model.Delta); err != nil {
			h.log.Error("failed to update counter", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if total, ok := h.storage.Counter(model.ID); ok {
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
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

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
		value, ok := h.storage.Gauge(model.ID)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		model.Value = &value
	case models.Counter:
		value, ok := h.storage.Counter(model.ID)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
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
	w.Write(data)
}
