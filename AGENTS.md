# Go Runtime Observer Instructions

Context for AI coding assistants. Complements README.md and CONTRIBUTING.md.

## Project Purpose

Instruments Go toolchains to capture runtime behavior (reflection, unsafe operations) by patching Go source during compilation. Does NOT modify user code - instrumentation happens in temporary directories.

## Security & Ethics

**CRITICAL**: This project is designed for knowledge gathering of runtime internals only. Instrumentation must never be used to create malformed or insecure Go toolchains. All modifications are for observation and logging purposes - the instrumented Go toolchain must maintain the same security properties and correctness as the original Go toolchain.

- Only add logging/wrapper functions
- Never modify core Go behavior or security checks
- Never introduce vulnerabilities or bypass safety mechanisms
- All changes must be transparent and observable

## Architecture

| Component                     | Purpose                                                                    | Key Files                        |
| ----------------------------- | -------------------------------------------------------------------------- | -------------------------------- |
| `cmd/install-instrumentation` | Patches Go package loader, applies AST transformations                     | `versions/vX_Y/config.go`        |
| `pkg/preprocessor`            | Classifies files (stdlib/dependency/user), injects instrumentation         | `config.go`, `ast_inject.go`     |
| `pkg/instrumentation`         | Wrapper functions for stdlib (minimal for reflect, full stdlib for crypto) | `instrumentlog/`, `unsafe/v1_*/` |

## Constraints

| Constraint             | Details                                                                                    | Why                                                                                     |
| ---------------------- | ------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------- |
| **Security**           | Observation only - never create insecure toolchains                                        | Maintains toolchain integrity and user trust                                            |
| **No stdlib imports**  | Cannot use `fmt`, `encoding/json`, `log`, `strconv` in reflect instrumentation             | Avoids import cycles - `fmt` depends on `reflect`, so cannot use `fmt` in reflect       |
| **General stdlib use** | Most stdlib packages (crypto, net, etc.) can use full stdlib (`fmt`, `encoding/hex`, etc.) | Only reflect and low-level packages (like `os` if causing cycles) need minimal logging  |
| **Version-specific**   | Each Go minor version has separate config (`versions/vX_Y/`)                               | Go's internal structure changes between versions                                        |
| **AST limitations**    | Only injects log calls at function call sites in stdlib                                    | AST tool is for stubbing log lines into existing code, not modifying signatures/structs |

## Key Files

| Path                                                           | Purpose                                   |
| -------------------------------------------------------------- | ----------------------------------------- |
| `cmd/install-instrumentation/main.go`                          | Entry point for instrumentation installer |
| `cmd/install-instrumentation/internal/versions/vX_Y/config.go` | Version-specific injection configs        |
| `cmd/install-instrumentation/internal/versions/versions.go`    | Version registry and lookup               |
| `pkg/preprocessor/config.go`                                   | Instrumentation function registry         |
| `pkg/preprocessor/ast_inject.go`                               | AST transformation logic                  |
| `pkg/instrumentation/instrumentlog/logger.go`                  | Logging functions (no stdlib)             |
| `.github/workflows/ci.yml`                                     | CI pipeline with `[skip-ci]` support      |
| `.githooks/commit-msg`                                         | Conventional commit validation            |
| `.github/go-versions.json`                                     | Supported Go versions for Docker builds   |

## Commands

| Category    | Command                                  | Purpose                         |
| ----------- | ---------------------------------------- | ------------------------------- |
| **CI**      | `make ci`                                | fmt + lint + test               |
| **Docker**  | `make docker-build`                      | Build instrumented Go container |
|             | `make dev-docker-run`                    | Build and run example app       |
|             | `make dev-docker-shell`                  | Interactive shell               |
| **Local**   | `make dev-clean-install-instrumented-go` | Full setup                      |
|             | `make dev-local-instrument`              | Copy instrumentation            |
|             | `make dev-local-build`                   | Rebuild Go                      |
|             | `make dev-local-test`                    | Test with example               |
| **Testing** | `make test`                              | Unit tests                      |
|             | `make test-coverage`                     | Coverage report                 |
|             | `make test-installation-compatibility`   | Test across Go versions         |

## Version Support

Supported: 1.19.x, 1.20.x, 1.21.x, 1.22.x, 1.23.x, 1.24.x

Adding new version requires:

1. `cmd/install-instrumentation/internal/versions/vX_Y/config.go`
2. Register in `versions/versions.go`
3. Update `.github/go-versions.json` if needed
4. Test: `make test-installation-compatibility GO_VERSION=X.Y.Z`

## Conventions

| Aspect         | Rule                                                                                                  |
| -------------- | ----------------------------------------------------------------------------------------------------- |
| **Planning**   | Remind user to review features/fixes with maintainer before starting work                             |
| **Code style** | No comments unless warranted, active voice, self-documenting, no emojis in docs or code               |
| **Commits**    | Conventional commits: `<type>(<scope>): <subject>`                                                    |
| **Types**      | `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`, `merge` |
| **Skip CI**    | Add `[skip-ci]` (still use conventional format)                                                       |
| **Merges**     | Squash commits only                                                                                   |
| **Pre-commit** | `make setup-hooks` → runs `fmt`, `lint`, `test`                                                       |

## Common Tasks

**Add new instrumented function:**

1. Create wrapper in `pkg/instrumentation/`
2. Register in `pkg/preprocessor/config.go`
3. Add injections in `versions/vX_Y/config.go`
4. Test: `make docker-build && make dev-docker-run`

**Add new Go version:**

1. Create `cmd/install-instrumentation/internal/versions/vX_Y/config.go`
2. Add to `SupportedVersions` in `versions.go`
3. Update `.github/go-versions.json`
4. Test: `make test-installation-compatibility GO_VERSION=X.Y.Z`

## Common Mistakes

| Mistake                                     | Why It Fails                                                      | Solution                                                               |
| ------------------------------------------- | ----------------------------------------------------------------- | ---------------------------------------------------------------------- |
| Importing stdlib in reflect instrumentation | Creates import cycle (`fmt` depends on `reflect`)                 | Use custom formatters in `instrumentlog/logger.go` (no stdlib)         |
| Assuming all packages need minimal logging  | Most stdlib packages can use `fmt`, `encoding/hex`, `strconv`     | Only reflect (and low-level packages with cycles) need minimal logging |
| Modifying AST beyond injecting log calls    | AST tool only stubs log lines into existing stdlib function calls | Only inject instrumentation log calls at function call sites           |
| Missing version registration                | Runtime lookup fails                                              | Always register in `versions/versions.go`                              |
| Forgetting test command                     | Changes may break without detection                               | Always run `make docker-build && make dev-docker-run`                  |
| Non-conventional commits                    | Pre-commit hook rejects                                           | Use format: `<type>(<scope>): <subject>`                               |

## Testing Expectations

Tests must pass completely. Coverage focuses on `pkg/preprocessor/` and `cmd/install-instrumentation/internal/` (excludes instrumentation wrappers). Integration tests verify Docker builds succeed and example app runs with instrumentation logging. Compatibility tests validate across all supported Go versions.
