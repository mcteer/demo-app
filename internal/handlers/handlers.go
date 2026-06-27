package handlers

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/demo-app/catalog-service/internal/db"
	"github.com/demo-app/catalog-service/internal/store"
	"github.com/go-chi/chi/v5"
)

//go:embed admin.html
var adminHTML []byte

type Handler struct {
	pool   *db.Pool
	store  *store.Store
	reload func(context.Context) error
}

func New(pool *db.Pool, reload func(context.Context) error) *Handler {
	return &Handler{
		pool:   pool,
		store:  store.New(pool.Underlying()),
		reload: reload,
	}
}

func (h *Handler) Healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Readyz(w http.ResponseWriter, r *http.Request) {
	if err := h.pool.Ready(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "not ready",
			"error":  err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *Handler) Admin(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(adminHTML)
}

func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.store.ListProducts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list products")
		return
	}
	if products == nil {
		products = []store.Product{}
	}
	writeJSON(w, http.StatusOK, products)
}

func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	product, err := h.store.GetProduct(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get product")
		return
	}

	writeJSON(w, http.StatusOK, product)
}

func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeProductInput(w, r)
	if !ok {
		return
	}

	product, err := h.store.CreateProduct(r.Context(), in)
	if err != nil {
		if errors.Is(err, store.ErrDuplicateSKU) {
			writeError(w, http.StatusConflict, "sku already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create product")
		return
	}

	writeJSON(w, http.StatusCreated, product)
}

func (h *Handler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	in, ok := decodeProductInput(w, r)
	if !ok {
		return
	}

	product, err := h.store.UpdateProduct(r.Context(), id, in)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		if errors.Is(err, store.ErrDuplicateSKU) {
			writeError(w, http.StatusConflict, "sku already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update product")
		return
	}

	writeJSON(w, http.StatusOK, product)
}

func (h *Handler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	if err := h.store.DeleteProduct(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete product")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseID(r *http.Request) (int, error) {
	return strconv.Atoi(chi.URLParam(r, "id"))
}

func decodeProductInput(w http.ResponseWriter, r *http.Request) (store.ProductInput, bool) {
	var in store.ProductInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return store.ProductInput{}, false
	}
	if in.Name == "" || in.SKU == "" || in.PriceCents < 0 {
		writeError(w, http.StatusBadRequest, "name, sku, and price_cents are required")
		return store.ProductInput{}, false
	}
	return in, true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
