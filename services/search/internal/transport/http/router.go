package http

import "net/http"

func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/search", h.HandleSearch)
	mux.HandleFunc("/index", h.HandleIndex)
	mux.HandleFunc("/stats", h.HandleStats)
	mux.HandleFunc("/health", h.HandleHealth)

	return mux
}
