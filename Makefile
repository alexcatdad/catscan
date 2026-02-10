.PHONY: build dev dev-backend dev-frontend test lint clean help

# Default target
help:
	@echo "CatScan Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build    - Build Svelte frontend and Go binary"
	@echo "  dev      - Run Go with frontend dev server"
	@echo "  test     - Run Go tests and svelte-check"
	@echo "  lint     - Run golangci-lint and biome check"
	@echo "  clean    - Remove build artifacts"

# Build Svelte frontend first, then Go binary
build:
	@echo "Building Svelte frontend..."
	cd frontend && bun run build
	@echo "Building Go binary..."
	go build -o ./bin/catscan ./cmd/catscan
	@echo "Build complete: ./bin/catscan"

# Run in development mode
dev:
	@echo "Starting development servers..."
	@make dev-backend & make dev-frontend

dev-backend:
	go run ./cmd/catscan

dev-frontend:
	cd frontend && bun run dev

# Run tests
test:
	@echo "Running Go tests..."
	go test -v -race ./...
	@echo "Running svelte-check..."
	cd frontend && bun run check

# Run linters
lint:
	@echo "Running golangci-lint..."
	golangci-lint run ./...
	@echo "Running biome check..."
	cd frontend && bunx biome check

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf ./bin
	rm -rf ./dist
	cd frontend && rm -rf node_modules .svelte-kit
	@echo "Clean complete"
