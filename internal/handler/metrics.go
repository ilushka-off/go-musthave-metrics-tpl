package handler

import (
	"net/http"
	"strconv"

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

	metricsType := r.PathValue("type")
	metricsName := r.PathValue("name")
	metricsValue := r.PathValue("value")

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
