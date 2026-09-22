package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/middleware"
	"go.uber.org/zap"
)

// NewRouter builds the chi.Router for the metrics server: it wires up
// logging, gzip and request-signing middleware, mounts the pprof debug
// endpoints under /debug, and registers all metric and health-check routes.
// p may be nil if the server was started without a database, in which case
// GET /ping responds with 503 without touching p.
func NewRouter(h *MetricsHandler, log *zap.Logger, p *PingHandler, key string) chi.Router {
	r := chi.NewRouter()
	r.Use(chimw.Recoverer)
	r.Use(chimw.StripSlashes)
	r.Use(middleware.Logger(log))
	r.Use(middleware.GzipDecompress())
	r.Use(middleware.GzipCompress())
	r.Use(middleware.Hash(key, log))
	if p != nil {
		r.Get("/ping", p.Ping)
	} else {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		})
	}
	r.Post("/update/{type}/{name}/{value}", h.Update)
	r.Post("/update", h.UpdateJSON)
	r.Get("/value/{type}/{name}", h.Value)
	r.Get("/", h.Index)
	r.Post("/value", h.ValueJSON)
	r.Post("/updates", h.UpdateBatch)
	r.Mount("/debug", chimw.Profiler())
	return r
}
