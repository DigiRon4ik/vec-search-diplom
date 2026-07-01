package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/marketplace/catalog/internal/domain"
	"github.com/marketplace/catalog/internal/usecase"
)

// Handler holds HTTP handlers for product endpoints.
type Handler struct {
	svc *usecase.ProductService
}

// NewHandler creates a new Handler.
func NewHandler(svc *usecase.ProductService) *Handler {
	return &Handler{svc: svc}
}

// HandleListProducts handles GET /products with ?ids=, ?limit=, ?offset=, ?category= params.
func (h *Handler) HandleListProducts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	q := r.URL.Query()

	if idsParam := q.Get("ids"); idsParam != "" {
		var ids []int64
		for _, s := range strings.Split(idsParam, ",") {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			id, parseErr := strconv.ParseInt(s, 10, 64)
			if parseErr != nil {
				writeError(w, http.StatusBadRequest, "invalid id in ids parameter")
				return
			}
			ids = append(ids, id)
		}
		products, err := h.svc.GetProductsByIDs(r.Context(), ids)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list products")
			return
		}
		if products == nil {
			products = []domain.Product{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"products": products})
		return
	}

	f := domain.ProductFilter{
		Category: q.Get("category"),
	}
	if v := q.Get("limit"); v != "" {
		f.Limit, _ = strconv.Atoi(v)
	}
	if v := q.Get("offset"); v != "" {
		f.Offset, _ = strconv.Atoi(v)
	}
	if f.Limit <= 0 {
		f.Limit = 24
	}

	products, total, err := h.svc.ListFiltered(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list products")
		return
	}
	if products == nil {
		products = []domain.Product{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"products": products,
		"total":    total,
		"offset":   f.Offset,
		"limit":    f.Limit,
	})
}

// HandleListCategories handles GET /categories.
func (h *Handler) HandleListCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := h.svc.ListCategories(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list categories")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"categories": cats})
}

// HandleGetProduct handles GET /products/{id}.
func (h *Handler) HandleGetProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/products/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	product, err := h.svc.GetProduct(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get product")
		return
	}
	if product == nil {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}

	writeJSON(w, http.StatusOK, product)
}

// HandleCreateProduct handles POST /products.
func (h *Handler) HandleCreateProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var p domain.Product
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.CreateProduct(r.Context(), &p); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create product")
		return
	}

	writeJSON(w, http.StatusCreated, p)
}

// HandleHealth handles GET /health.
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
