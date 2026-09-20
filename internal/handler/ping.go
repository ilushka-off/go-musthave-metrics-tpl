package handler

import (
	"database/sql"
	"net/http"

	"go.uber.org/zap"
)

// PingHandler serves the database health-check endpoint.
type PingHandler struct {
	db  *sql.DB
	log *zap.Logger
}

// NewPingHandler creates a PingHandler that checks connectivity to db.
func NewPingHandler(db *sql.DB, log *zap.Logger) *PingHandler {
	return &PingHandler{db: db, log: log}
}

// Ping handles GET /ping: it pings the configured database and responds with
// 200 on success or 500 if the connection check fails. A nil handler or a
// handler with no database configured responds with 503.
func (h *PingHandler) Ping(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.db == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	err := h.db.PingContext(r.Context())
	if err != nil {
		h.log.Error("failed to ping database", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)

}
