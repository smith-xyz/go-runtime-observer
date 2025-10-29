.PHONY: all help clean clean-all \
	dev-setup dev-local-instrument dev-local-build dev-local-test \
	dev-clean-install-instrumented-go dev-update-example-gomod \
	docker-build dev-docker-run dev-docker-shell docker-clean \
	vendor-deps test test-verbose test-coverage test-coverage-html \
	lint fmt ci

GO_VERSION ?= 1.23.0
GO_SRC_DIR := .dev-go-source/$(GO_VERSION)
BUILD_CMD  := $(GO_SRC_DIR)/go/bin/go build -C examples/app -a -o $(PWD)/examples/app/example-app .
DOCKER_ENV := -e GO_INSTRUMENT_UNSAFE=false -e GO_INSTRUMENT_REFLECT=true

all: docker-build

##@ Help

help:
	@echo "Go Runtime Observer - Development Workflow"
	@echo ""
	@echo "Current Go version: $(GO_VERSION)"
	@echo ""
	@echo "Local Development:"
	@echo "  make dev-setup                       Download Go source if needed"
	@echo "  make dev-local-instrument            Copy instrumentation to local Go source"
	@echo "  make dev-local-build                 Build instrumented Go locally"
	@echo "  make dev-local-test                  Test with local instrumented Go"
	@echo "  make dev-clean-instrumented-go       Clean instrumentation: setup + instrument"
	@echo "  make dev-clean-install-instrumented-go  Clean install: setup + instrument + build"
	@echo ""
	@echo "Docker Workflow:"
	@echo "  make docker-build                    Build instrumented Go container image"
	@echo "  make dev-docker-run                  Build and run example with Docker"
	@echo "  make dev-docker-shell                Interactive shell with Docker"
	@echo ""
	@echo "Testing:"
	@echo "  make test                            Run all unit tests"
	@echo "  make test-verbose                    Run tests with verbose output"
	@echo "  make test-coverage                   Run tests with coverage report"
	@echo "  make test-coverage-html              Generate HTML coverage report"
	@echo "  make test-installation-compatibility Test installation across Go versions"
	@echo ""
	@echo "Code Quality:"
	@echo "  make lint                            Run linter checks"
	@echo "  make fmt                             Format code"
	@echo "  make ci                              Run all CI checks (fmt, lint, test)"
	@echo ""
	@echo "Utilities:"
	@echo "  make vendor-deps                     Vendor dependencies for example app"
	@echo "  make dev-update-example-gomod        Update example app go.mod version"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean                           Remove build artifacts"
	@echo "  make clean-all                       Remove build artifacts + Go source"
	@echo "  make docker-clean                    Remove Docker artifacts and images"
	@echo ""
	@echo "Examples (override Go version):"
	@echo "  make dev-local-build GO_VERSION=1.23.1"
	@echo "  make docker-build GO_VERSION=1.24.0"

##@ Local Development

dev-setup:
	@mkdir -p $(GO_SRC_DIR)
	@if [ ! -d "$(GO_SRC_DIR)/go" ]; then \
		echo "Downloading Go $(GO_VERSION) source..."; \
		cd $(GO_SRC_DIR) && \
		curl -sL https://go.dev/dl/go$(GO_VERSION).src.tar.gz -o go.src.tar.gz && \
		tar -xzf go.src.tar.gz && \
		rm go.src.tar.gz; \
	fi

dev-local-instrument:
	@./scripts/install-instrumentation-to-go.sh $(GO_SRC_DIR)/go $(GO_VERSION)

dev-local-build:
	@echo "Building instrumented Go $(GO_VERSION)..."
	@cd $(GO_SRC_DIR)/go/src && \
		unset GO_INSTRUMENT_UNSAFE GO_INSTRUMENT_REFLECT && \
		GOROOT_BOOTSTRAP=$$(go env GOROOT) && \
		./make.bash

dev-local-test: clean vendor-deps
	@rm -f examples/app/example-app examples/app/*.log
	@GOROOT="$(CURDIR)/$(GO_SRC_DIR)/go" \
		PATH="$(CURDIR)/$(GO_SRC_DIR)/go/bin:$$PATH" \
		GOTOOLCHAIN=local \
		GO_INSTRUMENT_UNSAFE=true \
		GO_INSTRUMENT_REFLECT=true \
		$(BUILD_CMD)
	@INSTRUMENTATION_LOG_PATH=$(PWD)/examples/app/local-instrumentation.log ./examples/app/example-app
	@echo ""
	@echo "Instrumentation log:"
	@cat examples/app/local-instrumentation.log

dev-clean-go: clean-all dev-setup

dev-clean-instrumented-go: clean-all dev-setup dev-local-instrument

dev-clean-install-instrumented-go: clean-all dev-setup dev-local-instrument dev-local-build

##@ Docker Workflow

docker-build:
	@bash ./scripts/docker-install-instrumented-go.sh $(GO_VERSION)

dev-docker-run: docker-build dev-update-example-gomod vendor-deps
	@rm -f examples/app/example-app examples/app/docker-instrumentation.log
	@docker run --rm \
		-v $(PWD)/examples/app:/work \
		$(DOCKER_ENV) \
		instrumented-go:$(GO_VERSION) \
		build -o example-app .
	@docker run --rm \
		-v $(PWD)/examples/app:/work \
		$(DOCKER_ENV) \
		-e INSTRUMENTATION_LOG_PATH=/work/docker-instrumentation.log \
		--entrypoint /work/example-app \
		instrumented-go:$(GO_VERSION)
	@echo ""
	@echo "Instrumentation log:"
	@cat examples/app/docker-instrumentation.log

dev-docker-shell: docker-build
	@echo "Inside the container, your current directory is mounted at /work"
	@echo "Set INSTRUMENTATION_LOG_PATH to capture logs"
	@docker run --rm -it \
		-v $(PWD):/work \
		-e INSTRUMENTATION_LOG_PATH=/work/docker-instrumentation.log \
		--entrypoint /bin/bash \
		instrumented-go:$(GO_VERSION)

##@ Utilities

vendor-deps: dev-update-example-gomod
	@cd examples/app && go mod vendor

dev-update-example-gomod:
	@sed "s/{{GO_VERSION}}/$(GO_VERSION)/" examples/app/go.mod.template > examples/app/go.mod

##@ Testing

COVERAGE_PKGS := ./pkg/preprocessor/... ./cmd/install-instrumentation/internal/...

test:
	@go test ./pkg/... ./cmd/...

test-verbose:
	@go test -v ./pkg/... ./cmd/...

test-coverage:
	@go test -cover $(COVERAGE_PKGS)
	@echo ""
	@go test -coverprofile=coverage.out $(COVERAGE_PKGS) > /dev/null 2>&1
	@go tool cover -func=coverage.out | grep -E '^(github.com|total:)'
	@echo ""
	@echo "Note: Coverage excludes instrumentation wrappers (pkg/instrumentation/*)"

test-coverage-html: test-coverage
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Open coverage.html in your browser"

test-installation-compatibility:
	@./scripts/test-installation-compatibility.sh

##@ Code Quality

fmt:
	@gofmt -w -s $$(find . -name '*.go' -not -path './.dev-*' -not -path './examples/app/vendor/*')
	@echo "Code formatted"

lint:
	@go vet ./pkg/... ./cmd/...
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./pkg/... ./cmd/...; \
	else \
		echo "staticcheck not found. Install with: go install honnef.co/go/tools/cmd/staticcheck@latest"; \
		exit 1; \
	fi

ci: fmt lint test
	@echo ""
	@echo "All CI checks passed!"

##@ Cleanup

clean:
	@rm -rf bin/ examples/app/*.log examples/app/example-app examples/app/vendor
	@rm -f coverage.out coverage.html

clean-all: clean
	@rm -rf $(GO_SRC_DIR)/

docker-clean:
	@rm -f examples/app/example-app examples/app/docker-instrumentation.log
	@docker rmi instrumented-go:$(GO_VERSION) 2>/dev/null || true
