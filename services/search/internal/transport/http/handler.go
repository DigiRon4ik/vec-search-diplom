package http

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/marketplace/search/internal/domain"
	"github.com/marketplace/search/internal/usecase"
)

type Handler struct {
	indexSvc  *usecase.IndexService
	searchSvc *usecase.SearchService
}

func NewHandler(indexSvc *usecase.IndexService, searchSvc *usecase.SearchService) *Handler {
	return &Handler{indexSvc: indexSvc, searchSvc: searchSvc}
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
	if len(req.Vector) == 0 {
		http.Error(w, "vector is required", http.StatusBadRequest)
		return
	}

	results, err := h.searchSvc.Search(req.Vector, req.K)
	if err != nil {
		log.Printf("search error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, domain.SearchResponse{Results: results})
}

func (h *Handler) HandleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := h.indexSvc.BuildIndex(r.Context()); err != nil {
		log.Printf("index error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) HandleStats(w http.ResponseWriter, r *http.Request) {
	stats := h.searchSvc.Stats()
	stats.IsIndexing = h.indexSvc.IsIndexing()
	writeJSON(w, http.StatusOK, stats)
}

func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
