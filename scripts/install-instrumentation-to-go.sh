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

copy_files_simple() {
    local source_dir="$1"
    local target_dir="$2"
    
    find "$source_dir" -name "*.go" ! -name "*_test.go" -exec sh -c '
        rel="${1#'"$source_dir"'/}"
        target="$2/$rel"
        mkdir -p "$(dirname "$target")"
        cp "$1" "$target"
    ' _ {} "$target_dir" \;
}

copy_files_with_import_rewrite() {
    local source_dir="$1"
    local target_dir="$2"
    local import_pattern="$3"
    local import_replacement="$4"
    
    find "$source_dir" -name "*.go" ! -name "*_test.go" -exec sh -c '
        rel="${1#'"$source_dir"'/}"
        target_subdir="$2/$(dirname "$rel")"
        mkdir -p "$target_subdir"
        target="$2/$rel"
        
        sed "s|'"$import_pattern"'|'"$import_replacement"'|g" "$1" > "$target"
    ' _ {} "$target_dir" \;
}

copy_unsafe_file() {
    local target_file="$1"
    
    if [[ "$GO_VERSION" < "1.20" ]]; then
        UNSAFE_SOURCE="pkg/instrumentation/unsafe/v1_19/unsafe.go"
    else
        UNSAFE_SOURCE="pkg/instrumentation/unsafe/v1_20/unsafe.go"
    fi
    
    sed 's|github.com/smith-xyz/go-runtime-observer/pkg/instrumentation/instrumentlog|runtime_observe_instrumentation/instrumentlog|g' \
        "$UNSAFE_SOURCE" > "$target_file"
}

echo "Instrumenting Go ${GO_VERSION}..."
echo "Copying instrumentation files..."

mkdir -p "${INSTRUMENTATION_DIR}"/{instrumentlog,formatlog,correlation,unsafe,preprocessor}

copy_files_simple \
    "pkg/instrumentation/correlation" \
    "${INSTRUMENTATION_DIR}/correlation"

copy_files_with_import_rewrite \
    "pkg/instrumentation/instrumentlog" \
    "${INSTRUMENTATION_DIR}/instrumentlog" \
    "github.com/smith-xyz/go-runtime-observer/pkg/instrumentation/" \
    "runtime_observe_instrumentation/"

copy_files_simple \
    "pkg/instrumentation/formatlog" \
    "${INSTRUMENTATION_DIR}/formatlog"

copy_unsafe_file "${INSTRUMENTATION_DIR}/unsafe/unsafe.go"

copy_files_with_import_rewrite \
    "pkg/preprocessor" \
    "${INSTRUMENTATION_DIR}/preprocessor" \
    "github.com/smith-xyz/go-runtime-observer/pkg/preprocessor/" \
    "runtime_observe_instrumentation/preprocessor/"

echo "Building install-instrumentation..."
go build -o bin/install-instrumentation ./cmd/install-instrumentation

echo "Installing instrumentation..."
./bin/install-instrumentation -go-version="${GO_VERSION}" "${GO_SOURCE_DIR}"

echo "Done! Go ${GO_VERSION} is now instrumented."
