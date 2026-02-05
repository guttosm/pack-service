# ───────────────────────────────────────────────────────────────────────────────
# Project settings
# ───────────────────────────────────────────────────────────────────────────────
SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

APP_NAME            ?= pack-service
GO                  ?= go
GOLANGCI_LINT       ?= golangci-lint
SWAG                ?= $(shell $(GO) env GOPATH)/bin/swag
MOCKERY             ?= $(shell $(GO) env GOPATH)/bin/mockery

EXCLUDE_PKGS_REGEX ?= internal/domain/model|internal/domain/dto|internal/mocks

PKGS := $(shell $(GO) list ./... | grep -Ev '$(EXCLUDE_PKGS_REGEX)')

TEST_FLAGS       ?= -race -shuffle=on -count=1
TEST_PARALLEL    ?= 4
# Enable testcontainers reuse for faster integration tests
export TESTCONTAINERS_REUSE_ENABLED ?= true
COVER_PROFILE    ?= coverage.out
COVER_MODE       ?= atomic
UNIT_COVER_PROFILE ?= unit-coverage.out
INTEGRATION_COVER_PROFILE ?= integration-coverage.out

# ───────────────────────────────────────────────────────────────────────────────
# Helpers
# ───────────────────────────────────────────────────────────────────────────────
define print-target
	@printf "  \033[36m%-20s\033[0m %s\n" "$(1)" "$(2)"
endef

help: ## Show this help
	@echo "Make targets for $(APP_NAME):"
	@echo
	$(call print-target,install,           Download deps & install tools)
	$(call print-target,run,               Run app locally)
	$(call print-target,build,             Build binary)
	$(call print-target,fmt,               go fmt)
	$(call print-target,tidy,              go mod tidy)
	$(call print-target,lint,              Run golangci-lint)
	$(call print-target,swagger,           Generate Swagger docs)
	$(call print-target,godoc,             Start godoc server)
	$(call print-target,godoc-build,        Generate static godoc HTML)
	$(call print-target,mocks,             Generate mocks with mockery)
	$(call print-target,test,              Run ALL tests (unit + integration in parallel))
	$(call print-target,test-unit,         Run unit tests only)
	$(call print-target,test-integration,  Run integration tests only)
	$(call print-target,coverage,          Print coverage summary)
	$(call print-target,coverage-html,     Open HTML coverage report)
	$(call print-target,coverage-merge,    Merge unit and integration coverage)
	$(call print-target,docker-up,         Docker Compose up)
	$(call print-target,docker-down,       Docker Compose down)
	$(call print-target,docker-restart,    Restart Compose stack)
	$(call print-target,clean,             Clean artifacts)
	$(call print-target,vet,               Run go vet)
	$(call print-target,analyze,           Run all static analysis)
	@echo

# ───────────────────────────────────────────────────────────────────────────────
# Setup / Dev
# ───────────────────────────────────────────────────────────────────────────────
install: ## Download deps & install tools
	$(GO) mod download
	$(GO) install github.com/swaggo/swag/cmd/swag@latest
	$(GO) install github.com/vektra/mockery/v2@latest

run: ## Run locally without Docker
	@echo "Running $(APP_NAME)..."
	$(GO) run ./cmd/main.go

build: ## Build Go binary
	@echo "Building $(APP_NAME)..."
	CGO_ENABLED=0 $(GO) build -ldflags='-w -s' -o $(APP_NAME) ./cmd/main.go

fmt: ## Format code
	$(GO) fmt ./...

tidy: ## Tidy go.mod/go.sum
	$(GO) mod tidy

lint: ## Run static analysis (golangci-lint)
	$(GOLANGCI_LINT) run

swagger: ## Generate Swagger docs
	$(SWAG) init -g cmd/main.go --parseDependency --parseInternal -o docs

godoc: ## Start godoc server (http://localhost:6060)
	@echo "Starting godoc server at http://localhost:6060"
	godoc -http=:6060

godoc-build: ## Generate static godoc HTML
	@echo "Generating static godoc documentation..."
	@mkdir -p docs/godoc
	godoc -url=/pkg/github.com/guttosm/pack-service > docs/godoc/index.html || echo "Note: Install godoc with: go install golang.org/x/tools/cmd/godoc@latest"

mocks: ## Generate mocks with mockery
	$(MOCKERY)

# ───────────────────────────────────────────────────────────────────────────────
# Testing
# ───────────────────────────────────────────────────────────────────────────────
test: ## Run ALL tests (unit + integration in parallel)
	@echo "→ Running unit and integration tests in parallel..."
	@$(MAKE) -j$(TEST_PARALLEL) test-unit test-integration
	@echo "→ Merging coverage reports..."
	@$(MAKE) coverage-merge
	@echo "✅ All tests completed!"

test-unit: ## Run ONLY unit tests
	@echo "→ Running unit tests..."
	$(GO) test $(PKGS) $(TEST_FLAGS) -coverprofile=$(UNIT_COVER_PROFILE) -covermode=$(COVER_MODE)
	@echo "✅ Unit tests completed!"

test-integration: ## Run ONLY integration tests
	@echo "→ Running integration tests..."
	$(GO) test -tags=integration $(PKGS) -count=1 -coverprofile=$(INTEGRATION_COVER_PROFILE) -covermode=$(COVER_MODE)
	@echo "✅ Integration tests completed!"

bench: ## Run benchmarks
	$(GO) test ./internal/service -bench=. -benchmem -benchtime=3s

coverage: ## Show coverage summary
	@if [ -f $(COVER_PROFILE) ]; then \
		$(GO) tool cover -func=$(COVER_PROFILE); \
	else \
		echo "⚠️  Coverage file not found. Run 'make test' first."; \
		exit 1; \
	fi

coverage-html: ## Open HTML coverage report
	@if [ -f $(COVER_PROFILE) ]; then \
		$(GO) tool cover -html=$(COVER_PROFILE); \
	else \
		echo "⚠️  Coverage file not found. Run 'make test' first."; \
		exit 1; \
	fi

coverage-merge: ## Merge unit and integration coverage reports
	@echo "→ Merging coverage reports..."
	@if [ -f $(UNIT_COVER_PROFILE) ] && [ -f $(INTEGRATION_COVER_PROFILE) ]; then \
		echo "mode: $(COVER_MODE)" > $(COVER_PROFILE); \
		grep -h -v "^mode:" $(UNIT_COVER_PROFILE) $(INTEGRATION_COVER_PROFILE) >> $(COVER_PROFILE) || true; \
		echo "✅ Coverage reports merged into $(COVER_PROFILE)"; \
	elif [ -f $(UNIT_COVER_PROFILE) ]; then \
		cp $(UNIT_COVER_PROFILE) $(COVER_PROFILE); \
		echo "⚠️  Only unit coverage found. Copied to $(COVER_PROFILE)"; \
	elif [ -f $(INTEGRATION_COVER_PROFILE) ]; then \
		cp $(INTEGRATION_COVER_PROFILE) $(COVER_PROFILE); \
		echo "⚠️  Only integration coverage found. Copied to $(COVER_PROFILE)"; \
	else \
		echo "⚠️  No coverage files found. Run tests first."; \
		exit 1; \
	fi

# ───────────────────────────────────────────────────────────────────────────────
# Docker
# ───────────────────────────────────────────────────────────────────────────────
docker-build: ## Build Docker image
	docker build -t $(APP_NAME):latest .

docker-up: ## Compose up (build)
	docker compose up --build -d

docker-down: ## Compose down
	docker compose down

docker-restart: docker-down docker-up ## Restart Compose stack

docker-logs: ## Show container logs
	docker compose logs -f

# ───────────────────────────────────────────────────────────────────────────────
# Housekeeping
# ───────────────────────────────────────────────────────────────────────────────
clean: ## Clean compiled files and coverage artifacts
	rm -f $(APP_NAME) $(COVER_PROFILE) $(UNIT_COVER_PROFILE) $(INTEGRATION_COVER_PROFILE)
	rm -rf docs/

vet: ## Run go vet static analysis
	$(GO) vet ./...

analyze: vet lint ## Run all static analysis tools

.PHONY: help install run build fmt tidy lint swagger godoc godoc-build mocks \
        test test-unit test-integration bench coverage coverage-html \
        docker-build docker-up docker-down docker-restart docker-logs \
        clean vet analyze
