#!/bin/bash
set -euo pipefail

if [ -z "${1:-}" ]; then
    echo "Error: GO_VERSION is required"
    echo "Usage: $0 <GO_VERSION>"
    echo "Example: $0 1.23.0"
    exit 1
fi

GO_VERSION=$1

echo "Building instrumented Go ${GO_VERSION} container..."

cd "$(dirname "$0")/.."

docker build \
    --build-arg GO_VERSION=${GO_VERSION} \
    -t instrumented-go:${GO_VERSION} \
    -f build/Dockerfile .

echo "Successfully built instrumented-go:${GO_VERSION}"
echo ""
echo "Usage:"
echo "  docker run --rm instrumented-go:${GO_VERSION} version"
echo "  docker run --rm -v \$(pwd):/work instrumented-go:${GO_VERSION} build ./..."