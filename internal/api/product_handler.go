package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"Product_Inventory/internal/product"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

var skuRegex = regexp.MustCompile(`^[A-Z0-9]+$`)

var validStatuses = map[string]bool{
	"draft":    true,
	"active":   true,
	"archived": true,
}

type ProductHandler struct {
	queries *product.Queries
}

func NewProductHandler(queries *product.Queries) *ProductHandler {
	return &ProductHandler{queries: queries}
}

// Request and Response payloads

type CreateProductRequest struct {
	Name          string   `json:"name"`
	Sku           string   `json:"sku"`
	Description   *string  `json:"description,omitempty"`
	Price         *float64 `json:"price"`
	StockQuantity *int32   `json:"stock_quantity,omitempty"`
	Status        *string  `json:"status,omitempty"`
}

type UpdateProductRequest struct {
	Name          *string  `json:"name,omitempty"`
	Sku           *string  `json:"sku,omitempty"`
	Description   *string  `json:"description,omitempty"`
	Price         *float64 `json:"price,omitempty"`
	StockQuantity *int32   `json:"stock_quantity,omitempty"`
	Status        *string  `json:"status,omitempty"`
}

type ProductResponse struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Sku           string    `json:"sku"`
	Description   *string   `json:"description,omitempty"`
	Price         float64   `json:"price"`
	StockQuantity int32     `json:"stock_quantity"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Pagination struct {
	TotalItems int64 `json:"total_items"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

type ListProductsResponse struct {
	Data       []ProductResponse `json:"data"`
	Pagination Pagination        `json:"pagination"`
}

// Uniform Error Envelope Schema
type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// Helper to convert db Product model to ProductResponse JSON
func toProductResponse(p product.Product) ProductResponse {
	var desc *string
	if p.Description.Valid {
		desc = &p.Description.String
	}

	var price float64
	if f8, err := p.Price.Float64Value(); err == nil && f8.Valid {
		price = f8.Float64
	}

	var createdAt time.Time
	if p.CreatedAt.Valid {
		createdAt = p.CreatedAt.Time
	}

	var updatedAt time.Time
	if p.UpdatedAt.Valid {
		updatedAt = p.UpdatedAt.Time
	}

	return ProductResponse{
		ID:            p.ID,
		Name:          p.Name,
		Sku:           p.Sku,
		Description:   desc,
		Price:         price,
		StockQuantity: p.StockQuantity,
		Status:        p.Status,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// Uniform error response helper
func respondError(w http.ResponseWriter, status int, code, message string, details map[string]string) {
	respondJSON(w, status, ErrorResponse{
		Error:   code,
		Message: message,
		Details: details,
	})
}

// Decodes JSON with strict syntax and type validation
func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(dst)
	if err != nil {
		var syntaxErr *json.SyntaxError
		var unmarshalTypeErr *json.UnmarshalTypeError

		switch {
		case errors.As(err, &syntaxErr):
			respondError(w, http.StatusBadRequest, "bad_request", fmt.Sprintf("Malformed JSON syntax at offset %d", syntaxErr.Offset), nil)
		case errors.As(err, &unmarshalTypeErr):
			msg := fmt.Sprintf("Invalid value for field '%s': expected %s", unmarshalTypeErr.Field, unmarshalTypeErr.Type.String())
			details := map[string]string{unmarshalTypeErr.Field: msg}
			respondError(w, http.StatusBadRequest, "bad_request", msg, details)
		case errors.Is(err, io.EOF):
			respondError(w, http.StatusBadRequest, "bad_request", "Request body cannot be empty", nil)
		case strings.HasPrefix(err.Error(), "json: unknown field"):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			respondError(w, http.StatusBadRequest, "bad_request", fmt.Sprintf("Unknown field %s in request body", fieldName), map[string]string{fieldName: "unknown field"})
		default:
			respondError(w, http.StatusBadRequest, "bad_request", "Invalid JSON: "+err.Error(), nil)
		}
		return err
	}
	return nil
}

// CreateProduct godoc
// @Summary Create a product
// @Description Add a new product to the inventory.
// @Tags Products
// @Accept json
// @Produce json
// @Param product body CreateProductRequest true "Product details"
// @Success 201 {object} ProductResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /products [post]
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req CreateProductRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		return
	}

	// Enforce field validation rules before hitting the database
	details := make(map[string]string)

	if strings.TrimSpace(req.Name) == "" {
		details["name"] = "name is required and cannot be empty"
	}

	req.Sku = strings.TrimSpace(req.Sku)
	if req.Sku == "" {
		details["sku"] = "sku is required and cannot be empty"
	} else if !skuRegex.MatchString(req.Sku) {
		details["sku"] = "sku must contain only uppercase alphanumeric characters"
	}

	if req.Price == nil {
		details["price"] = "price is required"
	} else if *req.Price < 0 {
		details["price"] = "price must be greater than or equal to 0"
	}

	stockQty := int32(0)
	if req.StockQuantity != nil {
		if *req.StockQuantity < 0 {
			details["stock_quantity"] = "stock_quantity must be greater than or equal to 0"
		} else {
			stockQty = *req.StockQuantity
		}
	}

	status := "draft"
	if req.Status != nil && strings.TrimSpace(*req.Status) != "" {
		s := strings.TrimSpace(*req.Status)
		if !validStatuses[s] {
			details["status"] = "status must be one of: draft, active, archived"
		} else {
			status = s
		}
	}

	if len(details) > 0 {
		respondError(w, http.StatusBadRequest, "bad_request", "Validation failed", details)
		return
	}

	var desc pgtype.Text
	if req.Description != nil {
		desc = pgtype.Text{String: *req.Description, Valid: true}
	}

	var priceNum pgtype.Numeric
	if err := priceNum.Scan(fmt.Sprintf("%.2f", *req.Price)); err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "Invalid price format", map[string]string{"price": "unable to parse price"})
		return
	}

	created, err := h.queries.CreateProduct(r.Context(), product.CreateProductParams{
		Name:          req.Name,
		Sku:           req.Sku,
		Description:   desc,
		Price:         priceNum,
		StockQuantity: stockQty,
		Status:        status,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		// Map PostgreSQL unique violation (23505) to 409 Conflict
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			respondError(w, http.StatusConflict, "conflict", fmt.Sprintf("Product with SKU '%s' already exists", req.Sku), nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "internal_server_error", "Failed to create product: "+err.Error(), nil)
		return
	}

	respondJSON(w, http.StatusCreated, toProductResponse(created))
}

// ListProducts godoc
// @Summary List products
// @Description List products with pagination and optional filtering by status.
// @Tags Products
// @Produce json
// @Param page query int false "Page number (default 1)" default(1)
// @Param page_size query int false "Number of items per page (default 10, max 100)" default(10)
// @Param status query string false "Filter by status" Enums(draft, active, archived)
// @Success 200 {object} ListProductsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /products [get]
func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	page := 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		p, err := strconv.Atoi(pageStr)
		if err != nil || p < 1 {
			respondError(w, http.StatusBadRequest, "bad_request", "page must be a positive integer", nil)
			return
		}
		page = p
	}

	pageSize := 10
	limitStr := r.URL.Query().Get("page_size")
	if limitStr == "" {
		limitStr = r.URL.Query().Get("limit")
	}
	if limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err != nil || l < 1 {
			respondError(w, http.StatusBadRequest, "bad_request", "page_size must be a positive integer", nil)
			return
		}
		if l > 100 {
			l = 100
		}
		pageSize = l
	}

	var statusParam pgtype.Text
	if s := r.URL.Query().Get("status"); s != "" {
		if !validStatuses[s] {
			respondError(w, http.StatusBadRequest, "bad_request", "invalid status filter: must be draft, active, or archived", map[string]string{
				"status": "must be draft, active, or archived",
			})
			return
		}
		statusParam = pgtype.Text{String: s, Valid: true}
	}

	totalItems, err := h.queries.CountProducts(r.Context(), statusParam)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_server_error", "Failed to count products: "+err.Error(), nil)
		return
	}

	offset := (page - 1) * pageSize
	products, err := h.queries.ListProducts(r.Context(), product.ListProductsParams{
		Limit:  int32(pageSize),
		Offset: int32(offset),
		Status: statusParam,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_server_error", "Failed to fetch products: "+err.Error(), nil)
		return
	}

	data := make([]ProductResponse, 0, len(products))
	for _, p := range products {
		data = append(data, toProductResponse(p))
	}

	totalPages := 0
	if totalItems > 0 {
		totalPages = int(math.Ceil(float64(totalItems) / float64(pageSize)))
	}

	respondJSON(w, http.StatusOK, ListProductsResponse{
		Data: data,
		Pagination: Pagination{
			TotalItems: totalItems,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	})
}

// GetProductByID godoc
// @Summary Get a product by ID
// @Description Retrieve details of a single product by its primary key ID.
// @Tags Products
// @Produce json
// @Param id path int true "Product ID"
// @Success 200 {object} ProductResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /products/{id} [get]
func (h *ProductHandler) GetProductByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		respondError(w, http.StatusBadRequest, "bad_request", "Invalid ID format: id must be a positive integer", nil)
		return
	}

	p, err := h.queries.GetProductByID(r.Context(), id)
	if err != nil {
		// Map pgx.ErrNoRows to 404 Not Found
		if errors.Is(err, pgx.ErrNoRows) {
			respondError(w, http.StatusNotFound, "not_found", fmt.Sprintf("Product with ID %d not found", id), nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "internal_server_error", "Failed to get product: "+err.Error(), nil)
		return
	}

	respondJSON(w, http.StatusOK, toProductResponse(p))
}

// UpdateProduct godoc
// @Summary Update an existing product
// @Description Update editable fields of a product by its ID.
// @Tags Products
// @Accept json
// @Produce json
// @Param id path int true "Product ID"
// @Param product body UpdateProductRequest true "Editable product fields"
// @Success 200 {object} ProductResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /products/{id} [put]
func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		respondError(w, http.StatusBadRequest, "bad_request", "Invalid ID format: id must be a positive integer", nil)
		return
	}

	var req UpdateProductRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		return
	}

	// Field validations before hitting the database
	details := make(map[string]string)

	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		details["name"] = "name cannot be empty"
	}

	if req.Sku != nil {
		trimmed := strings.TrimSpace(*req.Sku)
		if trimmed == "" {
			details["sku"] = "sku cannot be empty"
		} else if !skuRegex.MatchString(trimmed) {
			details["sku"] = "sku must contain only uppercase alphanumeric characters"
		}
	}

	if req.Price != nil && *req.Price < 0 {
		details["price"] = "price must be greater than or equal to 0"
	}

	if req.StockQuantity != nil && *req.StockQuantity < 0 {
		details["stock_quantity"] = "stock_quantity must be greater than or equal to 0"
	}

	if req.Status != nil {
		trimmed := strings.TrimSpace(*req.Status)
		if !validStatuses[trimmed] {
			details["status"] = "status must be one of: draft, active, archived"
		}
	}

	if len(details) > 0 {
		respondError(w, http.StatusBadRequest, "bad_request", "Validation failed", details)
		return
	}

	// Fetch existing product to verify existence and merge partial updates
	existing, err := h.queries.GetProductByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondError(w, http.StatusNotFound, "not_found", fmt.Sprintf("Product with ID %d not found", id), nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "internal_server_error", "Failed to retrieve existing product: "+err.Error(), nil)
		return
	}

	name := existing.Name
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
	}

	sku := existing.Sku
	if req.Sku != nil {
		sku = strings.TrimSpace(*req.Sku)
	}

	desc := existing.Description
	if req.Description != nil {
		desc = pgtype.Text{String: *req.Description, Valid: true}
	}

	priceNum := existing.Price
	if req.Price != nil {
		if err := priceNum.Scan(fmt.Sprintf("%.2f", *req.Price)); err != nil {
			respondError(w, http.StatusBadRequest, "bad_request", "Invalid price format", map[string]string{"price": "unable to parse price"})
			return
		}
	}

	stockQty := existing.StockQuantity
	if req.StockQuantity != nil {
		stockQty = *req.StockQuantity
	}

	status := existing.Status
	if req.Status != nil {
		status = strings.TrimSpace(*req.Status)
	}

	updated, err := h.queries.UpdateProduct(r.Context(), product.UpdateProductParams{
		ID:            id,
		Name:          name,
		Sku:           sku,
		Description:   desc,
		Price:         priceNum,
		StockQuantity: stockQty,
		Status:        status,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		// Map PostgreSQL unique violation (23505) to 409 Conflict
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			respondError(w, http.StatusConflict, "conflict", fmt.Sprintf("Product with SKU '%s' already exists", sku), nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "internal_server_error", "Failed to update product: "+err.Error(), nil)
		return
	}

	respondJSON(w, http.StatusOK, toProductResponse(updated))
}

// DeleteProduct godoc
// @Summary Delete a product
// @Description Remove a product from the inventory by its ID.
// @Tags Products
// @Param id path int true "Product ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /products/{id} [delete]
func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		respondError(w, http.StatusBadRequest, "bad_request", "Invalid ID format: id must be a positive integer", nil)
		return
	}

	// Verify product exists before deleting; map pgx.ErrNoRows to 404 Not Found
	_, err = h.queries.GetProductByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondError(w, http.StatusNotFound, "not_found", fmt.Sprintf("Product with ID %d not found", id), nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "internal_server_error", "Failed to check product: "+err.Error(), nil)
		return
	}

	if err := h.queries.DeleteProduct(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_server_error", "Failed to delete product: "+err.Error(), nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
