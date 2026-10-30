# Product Inventory API 📦

A production-ready RESTful Product Inventory API built with **Go**, **Chi router**, **PostgreSQL**, and **sqlc**, containerized with **Docker & Docker Compose**, and documented with **Swagger / OpenAPI**.

---

## ✨ Features

- **High-Performance Routing**: Built with [Chi v5](https://github.com/go-chi/chi) with standard production middleware (RequestID, RealIP, Logger, Recoverer, Timeout).
- **Type-Safe Database Access**: Schema-first SQL compiled to typed Go code via [sqlc](https://sqlc.dev/) with [pgx/v5](https://github.com/jackc/pgx).
- **Pre-Database Validation**: Strict JSON body decoding and comprehensive field validation returning uniform error responses.
- **Robust Error Handling**: Standardized error envelope (`error`, `message`, `details`) with proper HTTP status codes (`400 Bad Request`, `404 Not Found`, `409 Conflict`).
- **Pagination & Filtering**: Offset/limit pagination with total item counts and status filtering (`draft`, `active`, `archived`).
- **Database Migrations & Idempotent Seeder**: Automated schema migration and idempotent database seeder (`ON CONFLICT (sku) DO NOTHING`).
- **Full Containerization**: Multi-stage `Dockerfile` (optimized binary size, runs as non-root user) and `docker-compose.yml` with PostgreSQL healthcheck probes and persistent volumes.
- **Interactive API Documentation**: Embedded Swagger UI accessible at `/swagger/index.html`.

---

## 🛠️ Tech Stack

- **Language:** Go (1.25+)
- **HTTP Router:** Chi v5
- **Database:** PostgreSQL 16
- **SQL Generator / Driver:** sqlc & pgx/v5
- **Containerization:** Docker & Docker Compose
- **API Docs:** Swagger (swaggo/http-swagger)

---

## 🚀 Quick Start with Docker (Recommended)

The easiest way to run the entire stack (PostgreSQL + API):

### 1. Clone the repository
```bash
git clone https://github.com/Bitu-hane/Product_Inventory.git
cd Product_Inventory
```

### 2. Start the services
```bash
make docker-up
# or: docker compose up -d --build
```
> The API service automatically waits for PostgreSQL's `pg_isready` healthcheck probe to pass before starting.

### 3. Seed sample data
```bash
make docker-seed
# or: docker compose run --rm seeder
```

### 4. Access the API & Swagger UI
- **Swagger Documentation:** [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)
- **Health Check:** [http://localhost:8080/health](http://localhost:8080/health)
- **Products API:** [http://localhost:8080/api/v1/products](http://localhost:8080/api/v1/products)

---

## 💻 Running Locally Without Docker

### Prerequisites
- Go 1.25+
- PostgreSQL running locally with database `product_inventory`

### Setup
```bash
# 1. Run migrations
psql -U postgres -d product_inventory -f migrations/000001_create_products_table.up.sql

# 2. Seed database
make seed
# or: go run ./cmd/seeder

# 3. Start the API server
make server
# or: go run ./cmd/api
```

---

## 📡 API Endpoints

| Method | Endpoint | Description | Status Codes |
| :--- | :--- | :--- | :--- |
| **POST** | `/api/v1/products` | Create a new product | `201`, `400`, `409` |
| **GET** | `/api/v1/products` | List products with pagination & filters | `200` |
| **GET** | `/api/v1/products/{id}` | Get product details by ID | `200`, `400`, `404` |
| **PUT** | `/api/v1/products/{id}` | Update an existing product | `200`, `400`, `404`, `409` |
| **DELETE** | `/api/v1/products/{id}` | Remove a product | `204`, `400`, `404` |
| **GET** | `/swagger/index.html` | Interactive Swagger API documentation | `200` |
| **GET** | `/health` | Service health status | `200` |

### Sample JSON Error Response
All errors return a consistent envelope:
```json
{
  "error": "bad_request",
  "message": "Validation failed",
  "details": {
    "name": "name is required and cannot be empty",
    "price": "price must be greater than or equal to 0",
    "sku": "sku must contain only uppercase alphanumeric characters"
  }
}
```

---

## 📋 Useful Makefile Commands

```bash
make docker-up     # Build and start all containers in background
make docker-down   # Stop all containers
make docker-seed   # Run database seeder inside Docker container
make server        # Run the API server locally
make seed          # Run the seeder locally
make sqlc          # Recompile SQL queries to Go with sqlc
make swagger       # Regenerate Swagger documentation
```

---

## 📄 License
This project is licensed under the [MIT License](LICENSE).
