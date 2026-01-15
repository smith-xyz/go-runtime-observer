#!/bin/bash
set -euo pipefail

GO_MINOR_VERSIONS=("1.19" "1.20" "1.21" "1.22" "1.23" "1.24" "1.25")
TEST_SPECIFIC_VERSION="${TEST_SPECIFIC_VERSION:-${1:-}}"
VERBOSE="${VERBOSE:-false}"
AUTO_FETCH_LATEST="${AUTO_FETCH_LATEST:-false}" # requires jq and curl
FAILED_VERSIONS=()
PASSED_VERSIONS=()

silent() {
    if [ "${VERBOSE}" = "true" ]; then
        "$@"
    else
        "$@" >/dev/null 2>&1 || true
    fi
}

run_test_for_version() {
    local GO_VERSION=$1
    echo "Testing Go ${GO_VERSION}..."
    
    local TEST_DIR=$(mktemp -d)
    trap "rm -rf ${TEST_DIR}" RETURN
    
    silent make clean
    silent make dev-update-example-gomod GO_VERSION=${GO_VERSION}
    
    if ! make docker-build GO_VERSION=${GO_VERSION} 2>&1 | tee "${TEST_DIR}/docker-build.log" >/dev/null; then
        echo "  FAIL: Container build failed"
        echo "  Last 10 lines of build output:"
        tail -10 "${TEST_DIR}/docker-build.log" | sed 's/^/    /'
        return 1
    fi
    
    if ! silent docker image inspect instrumented-go:${GO_VERSION}; then
        echo "  FAIL: Container image not found after build"
        return 1
    fi
    
    if ! docker run --rm \
        -v $(pwd)/examples/app:/work \
        -e GO_INSTRUMENT_UNSAFE=true \
        -e GO_INSTRUMENT_REFLECT=true \
        instrumented-go:${GO_VERSION} \
        build -o /work/test-app-${GO_VERSION} . 2>&1 | tee "${TEST_DIR}/app-build.log" | tail -1; then
        echo "  FAIL: App build failed"
        echo "  Build output:"
        cat "${TEST_DIR}/app-build.log" | sed 's/^/    /'
        return 1
    fi
    
    if ! docker run --rm \
        -v $(pwd)/examples/app:/work \
        -e INSTRUMENTATION_LOG_PATH=/work/test-${GO_VERSION}.log \
        --entrypoint /work/test-app-${GO_VERSION} \
        instrumented-go:${GO_VERSION} 2>&1 | tee "${TEST_DIR}/app-run.log" >/dev/null; then
        echo "  FAIL: App execution failed"
        echo "  Execution output:"
        cat "${TEST_DIR}/app-run.log" | sed 's/^/    /'
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
    if [ "${VERBOSE}" = "true" ]; then
        echo "Verbose mode enabled"
    fi
    echo ""


    if [ "$AUTO_FETCH_LATEST" = "true" ]; then
        echo "Auto-fetching latest patch versions..."
        SUPPORTED_VERSIONS=()
        
        version_list=$(curl -s 'https://go.dev/dl/?mode=json&include=all' | jq -r '.[].version')
        
        for minor in "${GO_MINOR_VERSIONS[@]}"; do
            latest=$(echo "$version_list" | grep "^go${minor}" | head -1 | sed 's/^go//' || true)
            
            if [ -n "$latest" ]; then
                SUPPORTED_VERSIONS+=("$latest")
                echo "  ✓ Go $minor -> $latest"
            else
                echo "  ✗ Warning: Could not find latest for Go $minor, using default"
                SUPPORTED_VERSIONS+=("$minor")
            fi
        done
    else
        SUPPORTED_VERSIONS=("1.19" "1.20" "1.21.0" "1.22.0" "1.23.0" "1.24.0" "1.25.0")
    fi
    
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

