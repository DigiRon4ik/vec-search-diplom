package http

import "net/http"

func NewRouter(h *Handler, frontendDir string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/search", h.HandleSearch)
	mux.HandleFunc("/api/reindex", h.HandleReindex)
	mux.HandleFunc("/api/products", h.HandleProducts)
	mux.HandleFunc("/api/categories", h.HandleCategories)
	mux.HandleFunc("/health", h.HandleHealth)
	mux.Handle("/", http.FileServer(http.Dir(frontendDir)))

	return mux
}
