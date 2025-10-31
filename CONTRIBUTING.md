# Contributing

## First-Time Setup

Before making changes, configure Git pre-commit hooks:

```bash
make setup-hooks
```

This automatically runs formatting, linting, and tests before each commit, catching issues early and ensuring code quality. The CI will verify hooks are configured when you open a PR.

**To skip hooks temporarily (not recommended):**

```bash
git commit --no-verify
```

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

When a new Go minor version is released:

1. Create version config:

```bash
# Create new file for minor version
touch cmd/install-instrumentation/internal/versions/v1_25/config.go
```

2. Define base configuration for the minor version:

```go
package v1_25

import "github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions/config"

func GetConfig() config.VersionConfig {
    return config.VersionConfig{
        Go:          "1.25",
        BaseVersion: "1.25.0",
        Notes:       "Base config for Go 1.25.x - works for most patches",
        Injections: []config.InjectionConfig{
            {
                Name:        "dependency",
                TargetFile:  "src/cmd/go/internal/load/pkg.go",
                Line:        905,  // Update to actual line number
                Description: "Inject after Happy: label in dependency resolution path",
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
            {
                Name:        "command_line",
                TargetFile:  "src/cmd/go/internal/load/pkg.go",
                Line:        3190,  // Update to actual line number
                Description: "Inject after ImportDir call in goFilesPackage for command-line files",
                Instrument: config.InstrumentCall{
                    Function: "InstrumentPackageFiles",
                    Args:     []string{"bp.GoFiles", "dir"},
                    Result:   []string{"bp.GoFiles", "dir"},
                },
                Reparse: config.ReparseCall{
                    Result:   []string{"bp", "err"},
                    Function: "ctxt.ImportDir",
                    Args:     []string{"dir", "0"},
                },
            },
        },
        Patches: []config.PatchConfig{
            {
                Name:        "buildvcs_default",
                TargetFile:  "src/cmd/go/internal/cfg/cfg.go",
                Description: "Disable VCS stamping by default to support temp directory instrumentation",
                Find:        `BuildBuildvcs      = "auto"`,
                Replace:     `BuildBuildvcs      = "false"`,
            },
        },
        Overrides: map[string]config.VersionOverride{},
    }
}
```

3. Register version:

```go
// cmd/install-instrumentation/internal/versions/versions.go
import "github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions/v1_25"

var SupportedVersions = map[string]config.VersionConfig{
    "1.24": v1_24.GetConfig(),
    "1.25": v1_25.GetConfig(),
}
```

4. Test:

```bash
make docker-build GO_VERSION=1.25.0
make dev-docker-run GO_VERSION=1.25.0
```

### Adding Patch-Specific Overrides

When a patch version shifts line numbers (e.g., Go 1.25.3 moves code):

```go
Overrides: map[string]config.VersionOverride{
    "1.25.3": {
        Injections: []config.InjectionOverride{
            {Name: "dependency", Line: 907},    // Only specify what changed
            {Name: "command_line", Line: 3195}, // Other fields come from base
        },
    },
},
```

Test the specific patch version:

```bash
TEST_SPECIFIC_VERSION=1.25.3 make test-installation-compatibility
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

## Instrumentation Strategies

This project uses **two different approaches** for instrumentation depending on the target:

### Decision Matrix: AST Injection vs Wrapper

| Criteria       | AST Injection                               | Wrapper Module                             |
| -------------- | ------------------------------------------- | ------------------------------------------ |
| **Target**     | Stdlib packages                             | User/dependency code                       |
| **Use When**   | Need to instrument methods on stdlib types  | Can replace package imports                |
| **Location**   | Modifies stdlib source in-place             | Writes to temp directory                   |
| **Example**    | `reflect.Value.Call()` method               | `unsafe.Add()` function                    |
| **Registry**   | `StdlibAST` map                             | `Instrumentation` map                      |
| **Limitation** | Stdlib only (needs internal package access) | Can't instrument methods on existing types |

### When to Use Each Approach

**Use AST Injection for:**

- Stdlib packages with methods to instrument (`reflect`, potentially `net/http`)
- Functions deeply integrated with stdlib internals
- Operations that ALL code (user + deps + stdlib) should log

**Use Wrapper Modules for:**

- Builtin packages (`unsafe` - can't AST inject into compiler builtins)
- User and dependency code
- Simple function replacements without method instrumentation

### Hybrid Approach (Advanced)

Some stdlib packages could theoretically use BOTH strategies:

- **AST injection** for their own methods (e.g., `encoding/json.Encoder.Encode`)
- **Wrapper** for their `unsafe` operations (if added to `SafeStdlibPackages`)

However, the current design keeps them separate:

- Stdlib packages in `StdlibAST` get AST injection ONLY (their `unsafe` calls stay as-is)
- Stdlib packages in `SafeStdlibPackages` get wrapper ONLY (if they use instrumented packages)
- User/dependency code always uses wrappers

**Rationale**: Stdlib's internal `unsafe` usage is implementation detail. We care about observing user/dependency unsafe operations.

## Adding Instrumented Functions

### Option 1: AST Injection (Stdlib Methods)

**When**: Instrumenting stdlib package methods (e.g., `http.Client.Do`)

1. **Update Registry** (`pkg/preprocessor/registry.go`):

```go
StdlibAST: map[string]StdlibASTInstrumentation{
    "reflect": { /* existing */ },
    "net/http": {
        PackageName: "net/http",
        Functions: []string{"Get", "Post"},
        Methods: []StdlibMethodInstrumentation{
            {
                ReceiverType: "Client",
                MethodNames:  []string{"Do", "Get", "Post"},
            },
        },
    },
},
```

2. **Test**: AST injection happens automatically at compile-time

```bash
make docker-build
make dev-docker-run
# Check logs for new operations
```

### Option 2: Wrapper Module (User/Dependency Code)

**When**: Instrumenting package-level functions

1. **Create Wrapper** (`pkg/instrumentation/crypto/sha256.go`):

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

2. **Update Registry** (`pkg/preprocessor/registry.go`):

```go
var DefaultRegistry = Registry{
    Instrumentation: map[string]InstrumentedPackage{
        "unsafe": { /* existing */ },
        "crypto/sha256": {
            Pkg:       "runtime_observe_instrumentation/crypto/sha256",
            Functions: []string{"Sum256"},
        },
    },
    // Optionally add to safe stdlib (if instrumenting stdlib usage)
    SafeStdlibPackages: []string{
        "encoding/json",
        "crypto/sha256",  // If stdlib can safely use this wrapper
    },
}
```

3. **Update install script** (`scripts/install-instrumentation-to-go.sh`):

Add logic to copy the new module to Go source during installation.

4. **Test**

```bash
# Rebuild and test
make docker-build
make dev-docker-run

# Check logs for new instrumentation
cat examples/app/docker-instrumentation.log
```

## Debugging

### Correlation Tracking

The correlation tracking system bridges call graph gaps between `MethodByName` and `Call` operations. See `docs/correlation-algorithm.md` for algorithm details.

**Enable debug logging:**

```bash
export INSTRUMENTATION_DEBUG_CORRELATION=true
export INSTRUMENTATION_DEBUG_LOG_PATH=/path/to/correlation-debug.log
```

Debug logs show:

- `RECORD`: Method name recorded with receiver pointer
- `GET`: Correlation lookup attempts and results
- Receiver pointer values used for matching

**Example debug output:**

```
RECORD: methodValuePtr=1374390599712 methodName=Add baton=1374390599712 seq=1
GET: baton=1374390599712 MATCH methodName=Add seq=1
```

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
go test ./pkg/instrumentation/instrumentlog/... -v
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

### Merging Pull Requests

PRs are required to be merged as squash commits. When merging PRs, maintain conventional commit format:

**Option 1: Squash and Merge via GitHub UI**

- Use GitHub's "Squash and merge" button (this is the only merge method available via GitHub UI)
- Edit the commit message to follow conventional commits format
- Example: `feat: add support for Go 1.24` or `fix(parser): correct version detection`
- This creates a single conventional commit on `main`

**Option 2: Merge Locally**

If you need more control over the merge process:

```bash
# Fetch the PR branch
git fetch origin pull/<PR_NUMBER>/head:pr-<PR_NUMBER>
git checkout main
git pull origin main

# Squash merge with a conventional commit message
git merge --squash pr-<PR_NUMBER>
git commit -m "feat: add support for Go 1.24"
git push origin main
```

## Release Process

### Publishing Docker Images

Docker images are automatically built and published to GHCR when you push tags.

**1. Update Go versions** (if needed):

Edit `.github/go-versions.json`:

```json
{
  "versions": ["1.19.13", "1.20.14", "1.21.13", "1.22.12", "1.23.12", "1.24.9"]
}
```

**2. Create and push a tag**:

```bash
# Tag the release
git tag v1.0.0
git push origin v1.0.0
```

**3. GitHub Actions builds images**:

For each Go version in `.github/go-versions.json`, it creates:

- `go1.24.9-v1.0.0` - Specific Go + framework version (on tagged releases)
- `go1.24.9-edge` - Specific Go with latest development (on main branch)
- `v1.0.0` - Latest Go with specific framework (on tagged releases, only for newest Go)
- `edge` - Latest Go with latest development (on main branch, only for newest Go)

**4. Verify on GHCR**:

Check https://github.com/smith-xyz/go-runtime-observer/pkgs/container/go-runtime-observer

**Note on image visibility**: First-time pushes create private packages by default. To make them public:

1. Go to the package page on GitHub
2. Click "Package settings"
3. Scroll to "Danger Zone"
4. Click "Change visibility" → "Public"

### Version Bump Checklist

- [ ] Update `go-versions.json` with tested Go patch versions
- [ ] Test locally: `TEST_SPECIFIC_VERSION=1.24.9 make test-installation-compatibility`
- [ ] Update README if adding/removing Go version support
- [ ] Create git tag following semver (v1.0.0, v1.1.0, v2.0.0)
- [ ] Push tag to trigger Docker build workflow
- [ ] Verify images built successfully in GitHub Actions

## Questions?

Open an issue for:

- Bug reports
- Feature requests
- Architecture questions
- Go version support requests
