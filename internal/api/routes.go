package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// JSONContentTypeMiddleware ensures JSON Content-Type header on API responses
func JSONContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// NewRouter sets up the Chi router with required middlewares and routes.
func NewRouter(h *ProductHandler) *chi.Mux {
	r := chi.NewRouter()

	// Standard Chi middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Custom timeout middleware
	r.Use(middleware.Timeout(30 * time.Second))

	// Health check route
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Subrouter for API v1 products
	r.Route("/api/v1/products", func(r chi.Router) {
		r.Use(JSONContentTypeMiddleware)
		r.Post("/", h.CreateProduct)
		r.Get("/", h.ListProducts)
		r.Get("/{id}", h.GetProductByID)
		r.Put("/{id}", h.UpdateProduct)
		r.Patch("/{id}", h.UpdateProduct)
		r.Delete("/{id}", h.DeleteProduct)
	})

	return r
}
