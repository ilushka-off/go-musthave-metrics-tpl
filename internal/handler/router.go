package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/middleware"
	"go.uber.org/zap"
)

func NewRouter(h *MetricsHandler, log *zap.Logger) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger(log))
	r.Use(middleware.GzipDecompress())
	r.Use(middleware.GzipCompress())
	r.Post("/update/{type}/{name}/{value}", h.Update)
	r.Post("/update/", h.UpdateJSON)
	r.Get("/value/{type}/{name}", h.Value)
	r.Post("/value/", h.ValueJSON)
	return r
}
