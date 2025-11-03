package preprocessor

import (
	"bytes"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/smith-xyz/go-runtime-observer/pkg/preprocessor/ast"
	"github.com/smith-xyz/go-runtime-observer/pkg/preprocessor/types"
)

func ProcessFile(filePath string, config Config) ([]byte, bool, error) {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read file: %w", err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		return nil, false, fmt.Errorf("failed to parse file: %w", err)
	}

	if config.Registry == nil {
		return src, false, nil
	}

	builder := ast.NewWrapperBuilder(file, config.Registry).
		ReplaceInstrumentedCalls().
		AddInstrumentedImports().
		RemoveUnusedImports()

	if !builder.IsModified() {
		return src, false, nil
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return nil, false, fmt.Errorf("failed to format AST: %w", err)
	}

	transformedContent := buf.Bytes()
	if _, err := parser.ParseFile(token.NewFileSet(), filePath, transformedContent, parser.ParseComments); err != nil {
		return nil, false, fmt.Errorf("transformed code is invalid Go: %w", err)
	}

	return transformedContent, true, nil
}

func ProcessFileInPlace(filePath string, config Config) error {
	// Early exit if instrumentation is not enabled or file should not be instrumented
	if !config.ShouldInstrument() || config.Registry == nil || !config.Registry.ShouldInstrument(filePath) {
		return nil
	}

	content, modified, err := ProcessFile(filePath, config)
	if err != nil {
		return fmt.Errorf("preprocessing failed for %s: %w", filePath, err)
	}

	if modified {
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}

		if f, err := os.Open(filePath); err == nil {
			_ = f.Sync()
			f.Close()
		}
	}

	return nil
}

func ProcessFileToTemp(originalPath string, config Config) (string, error) {
	isStdlib := config.Registry.IsStdLib(originalPath)

	// Step 1: Try AST instrumentation (stdlib only)
	if isStdlib {
		stdlibContent, stdlibModified, err := ProcessStdlibFile(originalPath, config.Registry)
		if err != nil {
			return originalPath, fmt.Errorf("stdlib AST failed: %w", err)
		}

		if stdlibModified {
			err := os.WriteFile(originalPath, stdlibContent, 0644)
			if err != nil {
				return originalPath, fmt.Errorf("failed to write stdlib to original path: %w", err)
			}

			return originalPath, nil
		}

		// If stdlib but NOT in SafeStdlibPackages, stop here
		// (e.g., reflect/abi.go - no AST match, not safe for wrappers)
		if !config.Registry.IsStdLibSafe(originalPath) {
			return originalPath, nil // No wrapper instrumentation
		}
	}

	content, modified, err := ProcessFile(originalPath, config)
	if err != nil {
		return originalPath, fmt.Errorf("preprocessing failed for %s: %w", originalPath, err)
	}

	if !modified {
		return originalPath, nil
	}

	// Step 3: Write result
	if isStdlib {
		// Safe stdlib with wrapper - write in-place
		err := os.WriteFile(originalPath, content, 0644)
		if err != nil {
			return originalPath, fmt.Errorf("error writing stdlib with wrapper to original path %s: %w", originalPath, err)
		}
		return originalPath, nil
	} else {
		tempPath, err := GetInstrumentedFilePath(originalPath, config.Registry)
		if err != nil {
			return originalPath, fmt.Errorf("failed to get temp path: %w", err)
		}

		if err := os.WriteFile(tempPath, content, 0644); err != nil {
			return originalPath, fmt.Errorf("failed to write temp file: %w", err)
		}

		return tempPath, nil
	}
}

func InstrumentPackageFiles(goFiles []string, pkgDir string) ([]string, string) {
	config, err := LoadConfigFromEnv()
	if err != nil || !config.ShouldInstrument() {
		return goFiles, pkgDir
	}

	if config.Registry == nil {
		return goFiles, pkgDir
	}

	var instrumentedDir string
	processedFiles := make(map[string]bool)

	for _, file := range goFiles {
		fullPath := filepath.Join(pkgDir, file)
		if !config.Registry.ShouldInstrument(fullPath) {
			continue
		}

		tempPath, err := ProcessFileToTemp(fullPath, config)
		if err != nil {
			continue
		}

		if tempPath != fullPath {
			if instrumentedDir == "" {
				moduleType := GetModuleType(fullPath, config.Registry)
				moduleDir, err := EnsureModuleTypeDir(moduleType)
				if err != nil {
					continue
				}
				instrumentedDir = filepath.Join(moduleDir, filepath.Base(pkgDir))
				if err := os.MkdirAll(instrumentedDir, 0755); err != nil {
					continue
				}
			}

			targetPath := filepath.Join(instrumentedDir, file)
			data, err := os.ReadFile(tempPath)
			if err != nil {
				continue
			}
			if err := os.WriteFile(targetPath, data, 0644); err != nil {
				continue
			}
			processedFiles[file] = true
		}
	}

	if instrumentedDir == "" {
		return goFiles, pkgDir
	}

	for _, file := range goFiles {
		if processedFiles[file] {
			continue
		}
		fullPath := filepath.Join(pkgDir, file)
		if !config.Registry.ShouldInstrument(fullPath) {
			continue
		}
		targetPath := filepath.Join(instrumentedDir, file)
		data, err := os.ReadFile(fullPath)
		if err == nil {
			_ = os.WriteFile(targetPath, data, 0644)
			processedFiles[file] = true
		}
	}

	entries, err := os.ReadDir(pkgDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, ".s") {
				continue
			}
			if processedFiles[name] {
				continue
			}
			fullPath := filepath.Join(pkgDir, name)
			if strings.HasSuffix(name, ".s") || config.Registry.ShouldInstrument(fullPath) {
				targetPath := filepath.Join(instrumentedDir, name)
				data, err := os.ReadFile(fullPath)
				if err == nil {
					_ = os.WriteFile(targetPath, data, 0644)
				}
			}
		}
	}

	return goFiles, instrumentedDir
}

func instrumentStdlibFile(filePath string, packageName string, functions []string, methods []types.StdlibMethodInstrumentation) ([]byte, bool, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read file: %w", err)
	}

	if bytes.Contains(content, []byte(InstrumentationMarker)) {
		return content, false, nil
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return nil, false, fmt.Errorf("failed to parse file: %w", err)
	}

	injector := ast.NewFileInjector(file, fset, packageName)

	if len(functions) > 0 {
		injector.InjectFunctions(functions)
	}

	if len(methods) > 0 {
		injector.InjectMethods(methods)
	}

	if !injector.IsModified() {
		return content, false, nil
	}

	injector.AddImport(InstrumentlogImportPath)

	result, err := injector.Render()
	if err != nil {
		return nil, false, err
	}

	return result, true, nil
}
