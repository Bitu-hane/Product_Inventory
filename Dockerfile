# Stage 1: Builder Stage
FROM golang:alpine AS builder

WORKDIR /app

# Install certificates
RUN apk --no-cache add ca-certificates tzdata

# Leverage Docker layer caching for dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy application source code
COPY . .

# Compile binary with size & performance optimizations (-ldflags="-s -w" strips debugging symbols)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/seeder ./cmd/seeder

# Stage 2: Final Minimal Runtime Stage
FROM alpine:3.21

# Install necessary certificates and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Security best practice: Run with a non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy compiled binaries from builder stage
COPY --from=builder /app/bin/api /usr/local/bin/api
COPY --from=builder /app/bin/seeder /usr/local/bin/seeder
COPY --from=builder /app/migrations /app/migrations

# Set correct ownership for application files
RUN chown -R appuser:appgroup /app /usr/local/bin/api /usr/local/bin/seeder

USER appuser

EXPOSE 8080

ENTRYPOINT ["api"]
