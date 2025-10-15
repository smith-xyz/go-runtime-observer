# Go Runtime Observer

Instruments Go's standard library to log runtime operations during execution.

**Why**: See what your code actually does at runtime, not just what's in imports. Useful for security analysis, CVE triage, and understanding dependencies.

## Quick Start

```bash
# Local: Build instrumented Go (~5 min one-time)
make dev-build-go
make dev-run

# Docker: Build and run
make dev-docker-run

# Change Go version
make dev-run GO_VERSION=1.21.0
make dev-docker-run GO_VERSION=1.20.0
```

## Example Output

```bash
cat examples/app/instrumentation.log
```

```json
{"operation":"ValueOf","caller":"main.main","file":"/work/main.go","line":21}
{"operation":"ValueOf","caller":"encoding/json.(*encodeState).marshal","file":"/usr/local/go/src/encoding/json/encode.go","line":298}
```

## Usage

```bash
# Interactive shells
make dev-shell          # Local instrumented Go
make dev-docker-shell   # Docker container

# In shell:
go build -a -o myapp .
INSTRUMENTATION_LOG_PATH=./myapp.log ./myapp
cat myapp.log
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development workflow and extending instrumentation.

## License

Apache 2.0
