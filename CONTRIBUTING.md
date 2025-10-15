# Contributing

## Workflows

### Local Development

```bash
make dev-instrument    # Generate instrumented code
make dev-build-go      # Build instrumented Go (~5 min)
make dev-run           # Run example app
```

### Docker

```bash
make docker-build      # Build container (~10-15 min)
make dev-docker-run    # Run example with Docker
```

## Project Structure

```
instrumenter/        # AST-based code injector
runtime/             # Logger (no stdlib dependencies)
build/Dockerfile     # Multi-stage container build
examples/app/        # Test application
```

## Adding Instrumentation

1. Create injector in `instrumenter/reflect/newfunction.go`
2. Register in `instrumenter/main.go` switch statement
3. Test with `make dev-instrument`

Example injector:

```go
func InjectNewFunction(fn *ast.FuncDecl) {
	logCall := &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent("instrumentlog"),
				Sel: ast.NewIdent("LogCall"),
			},
			Args: []ast.Expr{
				&ast.BasicLit{Kind: token.STRING, Value: `"NewFunction"`},
			},
		},
	}
	fn.Body.List = append([]ast.Stmt{logCall}, fn.Body.List...)
}
```

## Constraints

**Import Cycles**: Logger cannot use packages that import `reflect`:

- Safe: `os`, `runtime`, `sync`
- Avoid: `fmt`, `encoding/json`, `log`

**AST Flow**: Parse → Inspect → Inject → Print

## Debugging

```bash
# Compare instrumented vs original
diff .dev-go-source/1.23.0/go/src/reflect/value.go instrumented/reflect/value.go

# Check function names match in switch statement
grep "case \"" instrumenter/main.go
```

## Questions?

Contributions are welcome! Whether you're fixing bugs, adding features, or improving documentation:

- Open an issue to discuss ideas or report problems
- Submit a PR with your changes
- Ask questions about the codebase or instrumentation approach
