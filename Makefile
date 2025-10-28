.PHONY: all help clean clean-all \
	dev-setup dev-local-instrument dev-local-build dev-local-test \
	dev-clean-install-instrumented-go dev-update-example-gomod \
	docker-build dev-docker-run dev-docker-shell docker-clean \
	vendor-deps

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
	@echo "  make dev-clean-install-instrumented-go  Clean install: setup + instrument + build"
	@echo ""
	@echo "Docker Workflow:"
	@echo "  make docker-build                    Build instrumented Go container image"
	@echo "  make dev-docker-run                  Build and run example with Docker"
	@echo "  make dev-docker-shell                Interactive shell with Docker"
	@echo ""
	@echo "Utilities:"
	@echo "  make vendor-deps                     Vendor dependencies for example app"
	@echo "  make dev-update-example-gomod        Update example app go.mod version"
	@echo ""
	@echo "Installation Testing:"
	@echo "  make test-installation-compatibility   Tests installation and compatibility of instrumentation"
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
		echo "✓ Go source downloaded"; \
	else \
		echo "✓ Go source already exists"; \
	fi

dev-local-instrument:
	@echo "Copying instrumentation to local Go source..."
	@./scripts/install-instrumentation-to-go.sh $(GO_SRC_DIR)/go $(GO_VERSION)
	@echo "✓ Instrumentation copied to Go source"

dev-local-build:
	@echo "Building instrumented Go $(GO_VERSION) locally..."
	@cd $(GO_SRC_DIR)/go/src && \
		unset GO_INSTRUMENT_UNSAFE GO_INSTRUMENT_REFLECT && \
		GOROOT_BOOTSTRAP=$$(go env GOROOT) && \
		./make.bash
	@echo "✓ Instrumented Go built locally"

dev-local-test: clean vendor-deps
	@echo "Testing with local instrumented Go $(GO_VERSION)..."
	@rm -f examples/app/example-app examples/app/*.log
	@echo "Building example app..."
	@GOROOT="$(CURDIR)/$(GO_SRC_DIR)/go" \
		PATH="$(CURDIR)/$(GO_SRC_DIR)/go/bin:$$PATH" \
		GOTOOLCHAIN=local \
		GO_INSTRUMENT_UNSAFE=true \
		GO_INSTRUMENT_REFLECT=true \
		$(BUILD_CMD)
	@echo "Running example app..."
	@INSTRUMENTATION_LOG_PATH=$(PWD)/examples/app/local-instrumentation.log ./examples/app/example-app
	@echo ""
	@echo "Instrumentation log:"
	@cat examples/app/local-instrumentation.log
	@echo ""
	@echo "✓ Local test complete"

dev-clean-install-instrumented-go: clean-all dev-setup dev-local-instrument dev-local-build

##@ Docker Workflow

docker-build:
	@echo "Building instrumented Go $(GO_VERSION) container..."
	@bash ./scripts/docker-install-instrumented-go.sh $(GO_VERSION)

dev-docker-run: docker-build dev-update-example-gomod vendor-deps
	@echo "Building example app with Docker..."
	@rm -f examples/app/example-app examples/app/docker-instrumentation.log
	@docker run --rm \
		-v $(PWD)/examples/app:/work \
		$(DOCKER_ENV) \
		instrumented-go:$(GO_VERSION) \
		build -o example-app .
	@echo "Running example app..."
	@docker run --rm \
		-v $(PWD)/examples/app:/work \
		$(DOCKER_ENV) \
		-e INSTRUMENTATION_LOG_PATH=/work/docker-instrumentation.log \
		--entrypoint /work/example-app \
		instrumented-go:$(GO_VERSION)
	@echo ""
	@echo "Instrumentation log:"
	@cat examples/app/docker-instrumentation.log
	@echo ""
	@echo "✓ Docker run complete"

dev-docker-shell: docker-build
	@echo "Starting Docker shell with instrumented Go $(GO_VERSION)..."
	@echo ""
	@echo "Inside the container, your current directory is mounted at /work"
	@echo "Usage examples:"
	@echo "  go build -o myapp ."
	@echo "  go run main.go"
	@echo ""
	@echo "Note: Set INSTRUMENTATION_LOG_PATH to capture logs"
	@docker run --rm -it \
		-v $(PWD):/work \
		-e INSTRUMENTATION_LOG_PATH=/work/docker-instrumentation.log \
		--entrypoint /bin/bash \
		instrumented-go:$(GO_VERSION)

##@ Utilities

vendor-deps: dev-update-example-gomod
	@echo "Vendoring dependencies for example app..."
	@cd examples/app && go mod vendor

dev-update-example-gomod:
	@echo "Updating example app go.mod to Go $(GO_VERSION)..."
	@sed "s/{{GO_VERSION}}/$(GO_VERSION)/" examples/app/go.mod.template > examples/app/go.mod
	@echo "✓ Updated examples/app/go.mod to Go $(GO_VERSION)"

##@ Installation Testing

test-installation-compatibility:
	@echo "Running installation tests"
	@./scripts/test-installation-compatibility.sh

##@ Cleanup

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/ examples/app/*.log examples/app/example-app examples/app/vendor
	@echo "✓ Clean"

clean-all: clean
	@echo "Removing Go source..."
	@rm -rf $(GO_SRC_DIR)/
	@echo "✓ Clean all"

docker-clean:
	@echo "Cleaning Docker artifacts..."
	@rm -f examples/app/example-app examples/app/docker-instrumentation.log
	@docker rmi instrumented-go:$(GO_VERSION) 2>/dev/null || true
	@echo "✓ Docker clean"
