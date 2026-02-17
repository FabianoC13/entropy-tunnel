# EntropyTunnel Build — Production Makefile

VERSION ?= 0.1.0
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(DATE) -s -w"

GO       := go
GOTEST   := $(GO) test
GOBUILD  := $(GO) build
GOVET    := $(GO) vet

BIN_DIR  := bin
SERVER   := $(BIN_DIR)/entropy-server
CLIENT   := $(BIN_DIR)/entropy-client

# Build tags for Xray-core integration
TAGS ?= ""

.PHONY: all build server client test lint clean docker release gui help

all: build ## Build everything

build: server client ## Build server and client binaries

server: ## Build server binary
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -tags "$(TAGS)" -o $(SERVER) ./cmd/entropy-server

client: ## Build client binary
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -tags "$(TAGS)" -o $(CLIENT) ./cmd/entropy-client

server-xray: ## Build server with real xray-core
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -tags "xray" -o $(SERVER) ./cmd/entropy-server

client-xray: ## Build client with real xray-core
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -tags "xray" -o $(CLIENT) ./cmd/entropy-client

test: ## Run all tests
	$(GOTEST) -v -count=1 -race ./...

test-short: ## Run tests without race detector
	$(GOTEST) -v -count=1 ./...

test-cover: ## Run tests with coverage
	$(GOTEST) -v -count=1 -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint: ## Run linters
	$(GOVET) ./...
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || echo "golangci-lint not installed"

clean: ## Clean build artifacts
	rm -rf $(BIN_DIR) coverage.out coverage.html
	rm -rf gui/node_modules gui/out

docker: ## Build Docker image
	docker build -t entropy-tunnel:$(VERSION) .
	docker tag entropy-tunnel:$(VERSION) entropy-tunnel:latest

docker-compose: ## Run with docker-compose
	docker compose up --build -d

gui-install: ## Install GUI dependencies
	cd gui && npm install

gui-dev: ## Run GUI in development mode
	cd gui && npm install && npm start

gui: ## Build Electron GUI distribution packages
	cd gui && npm install && npm run make

gui-release: build gui ## Build CLI binaries + GUI installers (full desktop release)
	@echo "Desktop release ready in gui/out/"

generate-keys: ## Generate Reality x25519 keys + UUID
	@bash scripts/generate-keys.sh

# Cross-compilation targets
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

release: clean ## Build release binaries for all platforms
	@mkdir -p $(BIN_DIR)/release
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) \
		-o $(BIN_DIR)/release/entropy-server-$${platform%/*}-$${platform#*/}$$([ $${platform%/*} = windows ] && echo .exe) \
		./cmd/entropy-server; \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) \
		-o $(BIN_DIR)/release/entropy-client-$${platform%/*}-$${platform#*/}$$([ $${platform%/*} = windows ] && echo .exe) \
		./cmd/entropy-client; \
	done
	@echo "Release binaries in $(BIN_DIR)/release/"

checksums: release ## Generate SHA256 checksums
	cd $(BIN_DIR)/release && shasum -a 256 * > checksums.txt
	@echo "Checksums: $(BIN_DIR)/release/checksums.txt"

deploy-test: build docker ## Deploy test stack (Docker Compose + health check)
	@bash scripts/deploy-test.sh

version: ## Show version info
	@echo "EntropyTunnel v$(VERSION) ($(COMMIT)) built $(DATE)"

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
