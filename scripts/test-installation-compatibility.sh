#!/bin/bash
set -euo pipefail

SUPPORTED_VERSIONS=("1.19" "1.23.0")
TEST_SPECIFIC_VERSION="${1:-}"
FAILED_VERSIONS=()
PASSED_VERSIONS=()

run_test_for_version() {
    local GO_VERSION=$1
    echo "Testing Go ${GO_VERSION}..."
    
    local TEST_DIR=$(mktemp -d)
    trap "rm -rf ${TEST_DIR}" RETURN
    
    if ! make docker-build GO_VERSION=${GO_VERSION} >/dev/null 2>&1; then
        echo "  FAIL: Container build failed"
        return 1
    fi
    
    if ! docker run --rm \
        -v $(pwd)/examples/app:/work \
        -e GO_INSTRUMENT_UNSAFE=true \
        -e GO_INSTRUMENT_REFLECT=true \
        instrumented-go:${GO_VERSION} \
        build -o /work/test-app-${GO_VERSION} . >/dev/null 2>&1; then
        echo "  FAIL: App build failed"
        return 1
    fi
    
    if ! docker run --rm \
        -v $(pwd)/examples/app:/work \
        -e INSTRUMENTATION_LOG_PATH=/work/test-${GO_VERSION}.log \
        --entrypoint /work/test-app-${GO_VERSION} \
        instrumented-go:${GO_VERSION} >/dev/null 2>&1; then
        echo "  FAIL: App execution failed"
        rm -f examples/app/test-app-${GO_VERSION}
        return 1
    fi
    
    local LOG_PATH="examples/app/test-${GO_VERSION}.log"
    
    if [ ! -f "${LOG_PATH}" ]; then
        echo "  FAIL: Log file not created"
        rm -f examples/app/test-app-${GO_VERSION}
        return 1
    fi
    
    local LOG_LINES=$(wc -l < "${LOG_PATH}" | tr -d ' ')
    if [ "$LOG_LINES" -lt 1 ]; then
        echo "  FAIL: Log file is empty"
        rm -f examples/app/test-app-${GO_VERSION} "${LOG_PATH}"
        return 1
    fi
    
    if ! grep -q "ValueOf" "${LOG_PATH}"; then
        echo "  FAIL: Expected operations not found"
        rm -f examples/app/test-app-${GO_VERSION} "${LOG_PATH}"
        return 1
    fi
    
    echo "  PASS: ${LOG_LINES} log entries captured"
    
    rm -f examples/app/test-app-${GO_VERSION} "${LOG_PATH}"
    return 0
}

main() {
    echo "Testing installation compatibility..."
    echo ""
    
    local VERSIONS_TO_TEST=()
    
    if [ -n "${TEST_SPECIFIC_VERSION}" ]; then
        VERSIONS_TO_TEST=("${TEST_SPECIFIC_VERSION}")
    else
        VERSIONS_TO_TEST=("${SUPPORTED_VERSIONS[@]}")
    fi
    
    for version in "${VERSIONS_TO_TEST[@]}"; do
        if run_test_for_version "${version}"; then
            PASSED_VERSIONS+=("${version}")
        else
            FAILED_VERSIONS+=("${version}")
        fi
    done
    
    echo ""
    echo "Results: ${#PASSED_VERSIONS[@]}/${#VERSIONS_TO_TEST[@]} passed"
    
    if [ ${#FAILED_VERSIONS[@]} -gt 0 ]; then
        echo "Failed: ${FAILED_VERSIONS[*]}"
        exit 1
    fi
}

main "$@"

