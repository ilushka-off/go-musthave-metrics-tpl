package handler

import "net/http"

func NewRouter(h *MetricsHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /update/{type}/{name}/{value}", h.Update)
	return mux
}
