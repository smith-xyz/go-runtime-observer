.PHONY: all build-instrumenter dev-setup dev-instrument dev-build-go dev-run dev-shell docker-build dev-docker-run dev-docker-shell docker-clean clean clean-all help

# Auto-detect Go version from system, or override with GO_VERSION=x.y.z
GO_VERSION ?= 1.23.0
GO_SRC_DIR := .dev-go-source/$(GO_VERSION)

all: build-instrumenter

# Build the instrumenter tool
build-instrumenter:
	@echo "Building instrumenter..."
	@go build -o bin/instrumenter ./instrumenter
	@echo "✓ Built bin/instrumenter"

# Download Go source if needed
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

# Run instrumenter to see instrumented artifacts
dev-instrument: build-instrumenter dev-setup
	@echo "Running instrumenter..."
	@rm -rf instrumented
	@./bin/instrumenter -src $(GO_SRC_DIR)/go -output instrumented
	@echo "  -> Copying runtime/ → instrumented/runtime/"
	@mkdir -p instrumented
	@cp -r runtime instrumented/
	@echo "✓ Instrumentation complete"

# Build instrumented Go toolchain locally (~5 min)
dev-build-go: build-instrumenter dev-setup dev-instrument
	@echo "Building instrumented Go toolchain..."
	@echo "  -> Copying instrumentation to Go..."
	@cp -r instrumented/* $(GO_SRC_DIR)/go/src/
	@echo "  -> Building Go (this takes ~5 minutes)..."
	@cd $(GO_SRC_DIR)/go/src && GOROOT_BOOTSTRAP=$$(go env GOROOT) ./make.bash > /dev/null 2>&1
	@echo "✓ Instrumented Go built at $(GO_SRC_DIR)/go/bin/go"

# Update example app go.mod to match current GO_VERSION
dev-update-example-gomod:
	@echo "Updating examples/app/go.mod to go $(GO_VERSION)..."
	@if [ ! -f examples/app/go.mod ]; then \
		echo "Generating examples/app/go.mod from template..."; \
	fi
	@sed 's/{{GO_VERSION}}/$(GO_VERSION)/' examples/app/go.mod.template > examples/app/go.mod
	@echo "✓ Updated go.mod"

# Run example with instrumented Go
dev-run: dev-update-example-gomod
	@echo "Building example with instrumented Go..."
	@rm -f examples/app/example-app examples/app/*.log
	@GOROOT=$(PWD)/$(GO_SRC_DIR)/go \
	PATH=$(PWD)/$(GO_SRC_DIR)/go/bin:$$PATH \
	GOTOOLCHAIN=local \
	$(GO_SRC_DIR)/go/bin/go build -C examples/app -a -o $(PWD)/examples/app/example-app .
	@echo "Running example..."
	@INSTRUMENTATION_LOG_PATH=$(PWD)/examples/app/instrumentation.log ./examples/app/example-app

# Drop into a shell with instrumented Go configured
dev-shell:
	@echo "Starting shell with instrumented Go..."
	@echo "GOROOT: $(PWD)/$(GO_SRC_DIR)/go"
	@echo "Usage: go build -a -o myapp /path/to/your/app.go"
	@echo "       INSTRUMENTATION_LOG_PATH=./myapp.log ./myapp"
	@GOROOT=$(PWD)/$(GO_SRC_DIR)/go \
	PATH=$(PWD)/$(GO_SRC_DIR)/go/bin:$$PATH \
	GOTOOLCHAIN=local \
	bash

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/ instrumented/ examples/app/*.log examples/app/example-app
	@echo "✓ Clean"

# Clean everything including downloaded Go source
clean-all: clean
	@echo "Removing Go source..."
	@rm -rf $(GO_SRC_DIR)/
	@echo "✓ Clean all"

# Build instrumented Go container image
docker-build:
	@echo "Building instrumented Go $(GO_VERSION) container..."
	@bash ./scripts/docker-build-instrumented-go.sh $(GO_VERSION)
	
# Build and run example app with Docker
dev-docker-run: docker-build dev-update-example-gomod
	@echo "Building example app with Docker..."
	@rm -f examples/app/example-app examples/app/docker-instrumentation.log
	@docker run --rm \
		-v $(PWD)/examples/app:/work \
		instrumented-go:$(GO_VERSION) \
		build -o example-app .
	@echo "Running example app..."
	@docker run --rm \
		-v $(PWD)/examples/app:/work \
		-e INSTRUMENTATION_LOG_PATH=/work/docker-instrumentation.log \
		--entrypoint /work/example-app \
		instrumented-go:$(GO_VERSION)
	@echo ""
	@echo "Instrumentation log:"
	@cat examples/app/docker-instrumentation.log
	@echo ""
	@echo "✓ Docker run complete"

# Interactive shell with instrumented Go via Docker
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

# Clean Docker-specific artifacts
docker-clean:
	@echo "Cleaning Docker artifacts..."
	@rm -f examples/app/example-app examples/app/docker-instrumentation.log
	@docker rmi instrumented-go:$(GO_VERSION) 2>/dev/null || true
	@echo "✓ Docker clean"

help:
	@echo "Go Runtime Observer"
	@echo ""
	@echo "Current Go version: $(GO_VERSION)"
	@echo ""
	@echo "Local Development:"
	@echo "  make build-instrumenter    Build the AST instrumenter tool"
	@echo "  make dev-instrument        Generate instrumented artifacts"
	@echo "  make dev-build-go          Build instrumented Go (~5 min)"
	@echo "  make dev-run               Run example with local instrumented Go"
	@echo "  make dev-shell             Interactive shell with local instrumented Go"
	@echo ""
	@echo "Docker Workflow (Recommended for CI/Testing):"
	@echo "  make docker-build          Build instrumented Go container image"
	@echo "  make dev-docker-run        Build and run example with Docker"
	@echo "  make dev-docker-shell      Interactive shell with Docker"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean                 Remove build artifacts"
	@echo "  make docker-clean          Remove Docker artifacts and images"
	@echo "  make clean-all             Remove everything including Go source"
	@echo ""
	@echo "Examples (change Go version):"
	@echo "  make dev-run GO_VERSION=1.21.0"
	@echo "  make dev-docker-run GO_VERSION=1.20.0"
