# ── Stage 1: Build ────────────────────────────────────────────────────
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install dependencies first (cacheable layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy source & build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /sains-api ./cmd/server/

# ── Stage 2: Runtime ──────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /sains-api .

# Copy templates & static files for admin dashboard
COPY --from=builder /app/internal/admin/templates ./internal/admin/templates
COPY --from=builder /app/static ./static
COPY --from=builder /app/templates ./templates

EXPOSE 8080

CMD ["./sains-api"]
