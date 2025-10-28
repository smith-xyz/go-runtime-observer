#!/bin/bash
set -euo pipefail

if [ -z "${1:-}" ]; then
    echo "Usage: $0 <GO_SOURCE_DIR> [GO_VERSION]"
    exit 1
fi

GO_SOURCE_DIR=$1
GO_VERSION=${2:-}

if [ ! -d "${GO_SOURCE_DIR}/src" ]; then
    echo "ERROR: Not a valid Go source tree (missing src directory)"
    exit 1
fi

if [ -z "${GO_VERSION}" ]; then
    if [ -f "${GO_SOURCE_DIR}/VERSION" ]; then
        GO_VERSION=$(cat "${GO_SOURCE_DIR}/VERSION" | sed 's/^go//')
    else
        echo "ERROR: GO_VERSION not provided and VERSION file not found"
        echo "Usage: $0 <GO_SOURCE_DIR> [GO_VERSION]"
        exit 1
    fi
fi

INSTRUMENTATION_DIR="${GO_SOURCE_DIR}/src/runtime_observe_instrumentation"

echo "Instrumenting Go ${GO_VERSION}..."
echo "Copying instrumentation files..."

# Copy all instrumentation files
mkdir -p "${INSTRUMENTATION_DIR}"/{instrumentlog,unsafe,reflect,preprocessor}

# Copy instrumentlog
cp pkg/instrumentation/instrumentlog/logger.go "${INSTRUMENTATION_DIR}/instrumentlog/"

# Copy unsafe with import transformation
sed 's|github.com/smith-xyz/go-runtime-observer/pkg/instrumentation/instrumentlog|runtime_observe_instrumentation/instrumentlog|g' \
    pkg/instrumentation/unsafe/unsafe.go > "${INSTRUMENTATION_DIR}/unsafe/unsafe.go"

# Copy reflect with import transformation  
sed 's|github.com/smith-xyz/go-runtime-observer/pkg/instrumentation/instrumentlog|runtime_observe_instrumentation/instrumentlog|g' \
    pkg/instrumentation/reflect/reflect.go > "${INSTRUMENTATION_DIR}/reflect/reflect.go"

# Copy preprocessor files
sed '/fmt\.Printf.*DEBUG/d' pkg/preprocessor/config.go > "${INSTRUMENTATION_DIR}/preprocessor/config.go"
cp pkg/preprocessor/{preprocessor.go,registry.go,tempdir.go} "${INSTRUMENTATION_DIR}/preprocessor/"

echo "Building install-instrumentation..."
go build -o bin/install-instrumentation ./cmd/install-instrumentation

echo "Installing instrumentation..."
./bin/install-instrumentation -go-version="${GO_VERSION}" "${GO_SOURCE_DIR}"

echo "Done! Go ${GO_VERSION} is now instrumented."
