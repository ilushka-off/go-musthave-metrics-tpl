package handler

import (
	"github.com/go-chi/chi/v5"
)

func NewRouter(h *MetricsHandler) chi.Router {
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", h.Update)
	r.Get("/value/{type}/{name}", h.Value)
	return r
}
