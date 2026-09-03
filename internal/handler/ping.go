package handler

import (
	"database/sql"
	"net/http"

	"go.uber.org/zap"
)

type PingHandler struct {
	db  *sql.DB
	log *zap.Logger
}

func NewPingHandler(db *sql.DB, log *zap.Logger) *PingHandler {
	return &PingHandler{db: db, log: log}
}

func (h *PingHandler) Ping(w http.ResponseWriter, r *http.Request) {
	err := h.db.PingContext(r.Context())
	if err != nil {
		h.log.Error("failed to ping database", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)

}
