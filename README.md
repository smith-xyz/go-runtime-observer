# Go Runtime Observer

Log what your Go code actually does at runtime - including calls to `reflect`, `unsafe`, and other standard library operations.

## What It Does

Instruments a Go toolchain to capture runtime behavior. When you build with an instrumented Go, it logs operations like:

- `reflect.ValueOf` calls and what triggered them
- `unsafe` pointer operations
- Any standard library function you configure

Your code stays completely untouched. The instrumentation happens during compilation in temporary directories.

## Why Use This

**Security Analysis**: See if your dependencies use reflection or unsafe operations you didn't know about.

**CVE Triage**: Quickly determine if vulnerable code paths are actually executed in your application.

**Dependency Auditing**: Understand what third-party packages really do at runtime, not just what they import.

## Quick Start

```bash
# Build and run the example
make dev-docker-run
```

Output shows calls from your code, dependencies, and the standard library:

```json
{"operation":"ValueOf","caller":"main.main","file":"/work/main.go","line":21}
{"operation":"ValueOf","caller":"gopkg.in/yaml.v3.unmarshal","file":"/tmp/go-runtime-observer-abc123/dependency/gopkg.in/yaml.v3/decode.go","line":156}
{"operation":"ValueOf","caller":"encoding/json.(*encodeState).marshal","file":"/tmp/go-runtime-observer-abc123/stdlib/encoding/json/encode.go","line":298}
```

## How It Works

1. Install instrumentation into a Go toolchain (local or Docker container)
2. Build your application with the instrumented Go
3. Run your application normally
4. Check the log file for captured operations

No changes to your code or build process required.

## Usage

### Docker (Recommended)

Build once, use for any project:

```bash
# Build instrumented Go container
make docker-build

# Use it to build your app
docker run --rm -v $(pwd):/work instrumented-go:1.23.0 build -o myapp .

# Run with logging enabled
INSTRUMENTATION_LOG_PATH=./runtime.log ./myapp

# View what happened
cat runtime.log
```

### Interactive Shell

```bash
# Start shell with instrumented Go
make dev-docker-shell

# Inside container
go build -o myapp .
INSTRUMENTATION_LOG_PATH=/work/runtime.log ./myapp
cat /work/runtime.log
```

### Local Installation

Install directly on your machine:

```bash
# Download and instrument Go 1.23.0
make dev-clean-install-instrumented-go

# Build and test
make dev-local-test
```

## Configuration

Enable instrumentation for specific packages:

```bash
# Enable reflect operations
GO_INSTRUMENT_REFLECT=true

# Enable unsafe operations
GO_INSTRUMENT_UNSAFE=true

# Set log path
INSTRUMENTATION_LOG_PATH=/path/to/log
```

## Supported Go Versions

Currently supports:

- Go 1.23.0 (and compatible patch versions like 1.23.1, 1.23.2)

Version matching is automatic - requesting `1.23.5` will use the `1.23.0` configuration if no specific `1.23.5` config exists.

## Adding New Functions to Instrument

1. Write a wrapper function in `pkg/instrumentation/`
2. Register it in `pkg/preprocessor/config.go`
3. Rebuild: `make docker-build`

Example:

```go
package sha256

import (
    "crypto/sha256"
    "runtime_observe_instrumentation/instrumentlog"
)

func Sum256(data []byte) [32]byte {
    instrumentlog.LogCall("crypto/sha256.Sum256")
    return sha256.Sum256(data)
}
```

## Adding Go Version Support

See [CONTRIBUTING.md](CONTRIBUTING.md) for instructions on adding new Go versions.

## How This Differs From Other Tools

**vs Traditional Instrumentation**: No code changes or agent setup required. Works with existing Go projects.

**vs Runtime Profilers**: Captures specific operations you care about, not just CPU/memory usage.

**vs Static Analysis**: Shows what actually runs, not just what could run based on imports.

## Project Structure

```
cmd/install-instrumentation/    Install instrumentation into Go toolchain
pkg/instrumentation/            Wrapper functions for stdlib packages
pkg/preprocessor/               Runtime instrumentation logic
examples/app/                   Example application
```

## Development

```bash
# Docker workflow
make docker-build       # Build instrumented container
make dev-docker-run     # Test with example app

# Local workflow
make dev-setup          # Download Go source
make dev-local-instrument
make dev-local-build
make dev-local-test
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed development guide.

## License

Apache 2.0
