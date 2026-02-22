.PHONY: dev build run clean test sqlc migrate-up migrate-down migrate-create

# ─── Development ──────────────────────────────
dev:
	GIN_MODE=debug go run cmd/server/main.go

# ─── Build ────────────────────────────────────
build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/sains-api cmd/server/main.go

run: build
	./bin/sains-api

# ─── Test ─────────────────────────────────────
test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.txt -covermode=atomic ./...
	go tool cover -html=coverage.txt -o coverage.html

# ─── Database ─────────────────────────────────
migrate-up:
	migrate -path db/migrations -database "$${DATABASE_URL}" -verbose up

migrate-down:
	migrate -path db/migrations -database "$${DATABASE_URL}" -verbose down 1

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir db/migrations -seq $$name

# ─── sqlc ─────────────────────────────────────
sqlc:
	sqlc generate

# ─── Clean ────────────────────────────────────
clean:
	rm -rf bin/ tmp/ coverage.txt coverage.html

# ─── Dependencies ─────────────────────────────
deps:
	go mod tidy
	go mod verify

# ─── Lint ─────────────────────────────────────
lint:
	golangci-lint run ./...

# ─── All ──────────────────────────────────────
all: deps lint test build
