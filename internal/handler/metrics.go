package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/repository"
)

type MetricsHandler struct {
	storage repository.Storage
}

func NewMetricsHandler(s repository.Storage) *MetricsHandler {
	return &MetricsHandler{storage: s}
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
		h.storage.UpdateGauge(metricsName, value)
	case models.Counter:
		value, err := strconv.ParseInt(metricsValue, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		h.storage.UpdateCounter(metricsName, value)
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
