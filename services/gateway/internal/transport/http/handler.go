package http

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/marketplace/gateway/internal/domain"
	"github.com/marketplace/gateway/internal/usecase"
)

type Handler struct {
	svc        *usecase.SearchService
	catalogURL string
	proxyHTTP  *http.Client
}

func NewHandler(svc *usecase.SearchService, catalogURL string) *Handler {
	return &Handler{
		svc:        svc,
		catalogURL: catalogURL,
		proxyHTTP:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (h *Handler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req domain.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Query == "" {
		http.Error(w, "query is required", http.StatusBadRequest)
		return
	}
	if req.K <= 0 {
		req.K = 10
	}

	results, err := h.svc.Search(r.Context(), req.Query, req.K)
	if err != nil {
		log.Printf("search error: %v", err)
		http.Error(w, "search failed", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func (h *Handler) HandleReindex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := h.svc.Reindex(r.Context()); err != nil {
		log.Printf("reindex error: %v", err)
		http.Error(w, "reindex failed", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) HandleProducts(w http.ResponseWriter, r *http.Request) {
	h.proxyCatalog(w, r, "/products")
}

func (h *Handler) HandleCategories(w http.ResponseWriter, r *http.Request) {
	h.proxyCatalog(w, r, "/categories")
}

func (h *Handler) proxyCatalog(w http.ResponseWriter, r *http.Request, path string) {
	target := h.catalogURL + path
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
	if err != nil {
		http.Error(w, "proxy error", http.StatusInternalServerError)
		return
	}

	resp, err := h.proxyHTTP.Do(req)
	if err != nil {
		log.Printf("catalog proxy error: %v", err)
		http.Error(w, "catalog unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
