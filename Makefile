.PHONY: build dev dev-backend dev-frontend test lint clean e2e test-server install uninstall help

# Installation paths
BINARY_PATH ?= ~/.local/bin/catscan
LAUNCHD_PATH = ~/Library/LaunchAgents/com.alexcatdad.catscan.plist
LOG_DIR = ~/.config/catscan/logs

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
	@echo "  install  - Build and install CatScan locally"
	@echo "  uninstall- Remove CatScan (preserves config)"

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

# Run E2E tests
e2e: build
	@echo "Running E2E tests..."
	cd frontend && bunx playwright test

# Run test server for E2E tests
test-server: build
	@echo "Starting test server on port 9527..."
	CATSCAN_TEST=1 ./bin/catscan --test

# Install CatScan
install: build
	@echo "Installing CatScan..."
	@# Create log directory
	@mkdir -p "$(LOG_DIR)"
	@# Copy binary
	@cp ./bin/catscan "$(BINARY_PATH)"
	@chmod +x "$(BINARY_PATH)"
	@# Generate and install launchd plist
	@sed -e 's|{{BINARY_PATH}}|$(BINARY_PATH)|g' \
		-e 's|{{HOME_DIR}}|$(HOME)|g' \
		com.alexcatdad.catscan.plist.template > "$(LAUNCHD_PATH)"
	@# Load the launchd agent
	@launchctl load "$(LAUNCHD_PATH)" 2>/dev/null || true
	@echo "CatScan installed successfully!"
	@echo "Binary: $(BINARY_PATH)"
	@echo "Launchd agent: $(LAUNCHD_PATH)"
	@echo "Logs: $(LOG_DIR)"
	@echo ""
	@echo "Configure your settings by:"
	@echo "1. Open http://localhost:7700 in your browser (or your configured port)"
	@echo "2. Click the settings icon (gear) to configure scan path and GitHub owner"

# Uninstall CatScan
uninstall:
	@echo "Uninstalling CatScan..."
	@# Unload the launchd agent
	@-launchctl unload "$(LAUNCHD_PATH)" 2>/dev/null || true
	@# Remove the plist
	@-rm "$(LAUNCHD_PATH)" 2>/dev/null || true
	@# Remove the binary
	@-rm "$(BINARY_PATH)" 2>/dev/null || true
	@echo "CatScan uninstalled."
	@echo "Note: Config and cache files in ~/.config/catscan/ were preserved."
