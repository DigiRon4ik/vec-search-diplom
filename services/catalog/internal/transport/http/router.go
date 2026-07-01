package http

import "net/http"

// NewRouter creates an http.Handler with all catalog routes registered.
func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", h.HandleHealth)
	mux.HandleFunc("/categories", h.HandleListCategories)

	mux.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.HandleListProducts(w, r)
		case http.MethodPost:
			h.HandleCreateProduct(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	mux.HandleFunc("/products/", func(w http.ResponseWriter, r *http.Request) {
		h.HandleGetProduct(w, r)
	})

	return mux
}
