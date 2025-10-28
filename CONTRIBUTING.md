# Contributing

## Development Workflow

### Docker (Recommended)

Fast iteration for testing changes:

```bash
make docker-build       # Build instrumented Go container
make dev-docker-run     # Build and run example app
make dev-docker-shell   # Interactive shell with instrumented Go
```

### Local Development

Full control for debugging:

```bash
# Initial setup (downloads and builds Go source)
make dev-clean-install-instrumented-go

# Incremental changes
make dev-local-instrument   # Copy instrumentation to Go source
make dev-local-build       # Rebuild Go toolchain
make dev-local-test        # Test with example app
```

## Project Architecture

### Core Components

**`cmd/install-instrumentation`**

- Patches Go's package loader to intercept compilation
- Version-specific injection configurations
- Applies patches to Go source code

**`pkg/preprocessor`**

- Classifies files (stdlib, dependency, user code)
- Rewrites files to temporary directories
- Injects instrumentation function calls
- Manages registry of safe packages and functions

**`pkg/instrumentation`**

- Wrapper functions for stdlib packages
- Logging without stdlib dependencies (avoids import cycles)
- Separate packages: `instrumentlog`, `reflect`, `unsafe`

### File Flow

```
User runs: go build

Go's pkg.go loader
    └─> preprocessor.InstrumentPackageFiles()
        ├─> Classify file (stdlib/dependency/user)
        ├─> Write to temp directory
        ├─> Parse AST
        ├─> Inject instrumentation calls
        ├─> Add imports
        └─> Return new file paths

Go compiler uses instrumented temp files
    └─> Original files untouched
```

## Adding Go Version Support

When a new Go version is released or line numbers shift:

1. Create version config:

```bash
# Create new file
touch cmd/install-instrumentation/internal/versions/v1_24/1_24_0.go
```

2. Define injection points:

```go
package v1_24

import "github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions/config"

func GetConfig() config.VersionConfig {
    return config.VersionConfig{
        Go:    "1.24.0",
        Notes: "Initial 1.24.0 support - verify pkg.go structure",
        Injections: []config.InjectionConfig{
            {
                Name:        "dependency",
                TargetFile:  "src/cmd/go/internal/load/pkg.go",
                Line:        905,  // Update to actual line number
                Description: "Inject after Happy: label",
                Instrument: config.InstrumentCall{
                    Function: "InstrumentPackageFiles",
                    Args:     []string{"data.p.GoFiles", "data.p.Dir"},
                    Result:   []string{"data.p.GoFiles", "data.p.Dir"},
                },
                Reparse: config.ReparseCall{
                    Result:   []string{"data.p", "data.err"},
                    Function: "buildContext.ImportDir",
                    Args:     []string{"data.p.Dir", "buildMode"},
                },
            },
            // Add command_line injection point
        },
        Patches: []config.PatchConfig{
            {
                Name:        "buildvcs_default",
                TargetFile:  "src/cmd/go/internal/cfg/cfg.go",
                Description: "Disable VCS stamping",
                Find:        `BuildBuildvcs      = "auto"`,
                Replace:     `BuildBuildvcs      = "false"`,
            },
        },
    }
}
```

3. Register version:

```go
// cmd/install-instrumentation/internal/versions/versions.go
import "github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions/v1_24"

var SupportedVersions = map[string]config.VersionConfig{
    "1.23.0": v1_23.GetConfig(),
    "1.24.0": v1_24.GetConfig(),
}
```

4. Test:

```bash
make docker-build GO_VERSION=1.24.0
make dev-docker-run GO_VERSION=1.24.0
```

### Finding Injection Points

Examine Go's package loader source:

```bash
# Download Go source
make dev-setup GO_VERSION=1.24.0

# Find the Happy: label and goFilesPackage function
grep -n "Happy:" .dev-go-source/1.24.0/go/src/cmd/go/internal/load/pkg.go
grep -n "func goFilesPackage" .dev-go-source/1.24.0/go/src/cmd/go/internal/load/pkg.go
```

Look for:

1. `Happy:` label in dependency resolution (usually around line 900)
2. `bp, err := ctxt.ImportDir(dir, 0)` in `goFilesPackage` (usually around line 3180)

## Adding Instrumented Functions

### 1. Create Wrapper Function

```go
// pkg/instrumentation/crypto/sha256.go
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

### 2. Update Registry

```go
// pkg/preprocessor/config.go
var DefaultConfig = InstrumentationConfig{
    // ... existing config ...
    Sha256: InstrumentationTarget{
        Pkg:       "runtime_observe_instrumentation/crypto/sha256",
        Functions: []string{"Sum256"},
    },
}

// Add to safe packages
var DefaultSafePackages = []string{
    // ... existing packages ...
    "crypto/sha256",
}
```

### 3. Test

```bash
# Rebuild and test
make docker-build
make dev-docker-run

# Check logs for new instrumentation
cat examples/app/docker-instrumentation.log
```

## Debugging

### Compare Original vs Instrumented

```bash
# View original Go source
cat .dev-go-source/1.23.0/go/src/cmd/go/internal/load/pkg.go

# Check injection was applied
grep -A5 "preprocessor.InstrumentPackageFiles" .dev-go-source/1.23.0/go/src/cmd/go/internal/load/pkg.go
```

### Test Instrumentation Logic

```bash
# Run unit tests
go test ./pkg/preprocessor/... -v
go test ./cmd/install-instrumentation/internal/... -v
```

### Check Temp Directory Structure

```bash
# Set verbose logging
export GO_INSTRUMENT_DEBUG=true

# Build and inspect temp files
make dev-local-test
ls -la /tmp/go-runtime-observer-*/
```

## Constraints

### Import Cycles

`pkg/instrumentation/instrumentlog` cannot import packages that use `reflect`:

- Safe: `os`, `runtime`, `sync`, `unsafe`
- Avoid: `fmt`, `encoding/json`, `log`, `strconv`

Use basic file I/O and string manipulation only.

### AST Limitations

The preprocessor rewrites AST nodes for function calls only. It does not:

- Modify function signatures
- Change struct definitions
- Rewrite type assertions
- Handle function values (only direct calls)

### Version Compatibility

Injection configs are Go version-specific. When Go's source structure changes:

- Line numbers shift
- Function signatures change
- File paths reorganize

Always test new Go versions before adding support.

## Testing Changes

### Unit Tests

```bash
# Test version matching
go test ./cmd/install-instrumentation/internal/versions/... -v

# Test preprocessing logic
go test ./pkg/preprocessor/... -v
```

### Integration Tests

```bash
# Full Docker build
make docker-build

# Run example app
make dev-docker-run

# Verify instrumentation log
cat examples/app/docker-instrumentation.log | jq .
```

### Manual Testing

```bash
# Interactive shell
make dev-docker-shell

# Inside container
cd /work/examples/app
go build -a -o test-app .
INSTRUMENTATION_LOG_PATH=/work/test.log ./test-app
cat /work/test.log
```

## Pull Request Guidelines

1. Test with at least one Go version (preferably 1.23.0)
2. Update documentation if adding new features
3. Add unit tests for new preprocessing logic
4. Run `make clean && make docker-build` successfully
5. Include example output if adding new instrumented functions

## Questions?

Open an issue for:

- Bug reports
- Feature requests
- Architecture questions
- Go version support requests
