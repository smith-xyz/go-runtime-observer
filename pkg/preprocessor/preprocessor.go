package preprocessor

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
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
	
	modified := false
	
	neededImports := make(map[string]string)
	
	// Step 1: check for instrumented function calls and replace them
	ast.Inspect(file, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok {
				packageName := ident.Name
				functionName := sel.Sel.Name
				
				// Check if this function is instrumented
				if config.Registry.IsInstrumented(packageName, functionName) {
					// Create alias name
					aliasName := packageName + "_instrumented"
					
					// Replace the call
					sel.X = &ast.Ident{Name: aliasName}
					neededImports[packageName] = aliasName
					modified = true
				}
			}
		}
		return true
	})
	
	// Step 2: Add instrumented imports for packages that need them
	for packageName, aliasName := range neededImports {
		if instrumentedPath, exists := config.Registry.GetInstrumentedImportPath(packageName); exists {
			ast.Inspect(file, func(n ast.Node) bool {
				if decl, ok := n.(*ast.GenDecl); ok && decl.Tok == token.IMPORT {
					hasImport := false
					for _, spec := range decl.Specs {
						if impSpec, ok := spec.(*ast.ImportSpec); ok && impSpec.Path != nil {
							if impSpec.Path.Value == fmt.Sprintf(`"%s"`, instrumentedPath) {
								hasImport = true
								break
							}
						}
					}
					
					if !hasImport {
						newImport := &ast.ImportSpec{
							Path: &ast.BasicLit{
								Kind:  token.STRING,
								Value: fmt.Sprintf(`"%s"`, instrumentedPath),
							},
							Name: &ast.Ident{Name: aliasName},
						}
						decl.Specs = append(decl.Specs, newImport)
						modified = true
					}
				}
				return true
			})
		}
	}
	
	// Step 3: Remove unused original imports only if ALL uses were instrumented
	for packageName := range neededImports {
		// Check if there are ANY non-instrumented uses of this package
		hasNonInstrumentedUses := false
		ast.Inspect(file, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok {
					if ident.Name == packageName {
						// Found a use of the original package name
						// Check if this specific function is NOT instrumented
						functionName := sel.Sel.Name
						if !config.Registry.IsInstrumented(packageName, functionName) {
							hasNonInstrumentedUses = true
							return false
						}
					}
				}
			}
			return true
		})
		
		// Only remove the import if ALL uses were instrumented
		if !hasNonInstrumentedUses {
			ast.Inspect(file, func(n ast.Node) bool {
				if decl, ok := n.(*ast.GenDecl); ok && decl.Tok == token.IMPORT {
					filteredSpecs := []ast.Spec{}
					for _, spec := range decl.Specs {
						if impSpec, ok := spec.(*ast.ImportSpec); ok && impSpec.Path != nil {
							importPath := impSpec.Path.Value
							// Check if this is the original package being instrumented
							// Match imports like "reflect" or "unsafe"
							if strings.HasPrefix(importPath, `"`) && strings.HasSuffix(importPath, `"`) {
								unquotedPath := importPath[1 : len(importPath)-1]
								if unquotedPath == packageName {
									// Skip this import as we've replaced all uses with instrumented version
									modified = true
									continue
								}
							}
						}
						filteredSpecs = append(filteredSpecs, spec)
					}
					decl.Specs = filteredSpecs
				}
				return true
			})
		}
	}
	
	if !modified {
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
	if !config.ShouldInstrument() || config.Registry == nil || !config.Registry.ShouldInstrument(originalPath) {
		return originalPath, nil
	}
	
	content, modified, err := ProcessFile(originalPath, config)
	if err != nil {
		return originalPath, fmt.Errorf("preprocessing failed for %s: %w", originalPath, err)
	}
	
	if !modified {
		return originalPath, nil
	}
	
	tempPath, err := GetInstrumentedFilePath(originalPath, config.Registry)
	if err != nil {
		return originalPath, fmt.Errorf("failed to get temp path: %w", err)
	}
	
	if err := os.WriteFile(tempPath, content, 0644); err != nil {
		return originalPath, fmt.Errorf("failed to write temp file: %w", err)
	}
	
	return tempPath, nil
}

func InstrumentPackageFiles(goFiles []string, pkgDir string) ([]string, string) {
	config, err := LoadConfigFromEnv()
	if err != nil || !config.ShouldInstrument() {
		return goFiles, pkgDir
	}
	
	var instrumentedDir string
	anyInstrumented := false
	
	for _, file := range goFiles {
		fullPath := filepath.Join(pkgDir, file)
		tempPath, err := ProcessFileToTemp(fullPath, config)
		if err != nil {
			continue
		}
		
		if tempPath != fullPath {
			anyInstrumented = true
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
		}
	}
	
	if !anyInstrumented {
		return goFiles, pkgDir
	}
	
	for _, file := range goFiles {
		targetPath := filepath.Join(instrumentedDir, file)
		if !fileExists(targetPath) {
			fullPath := filepath.Join(pkgDir, file)
			data, err := os.ReadFile(fullPath)
			if err == nil {
				_ = os.WriteFile(targetPath, data, 0644)
			}
		}
	}
	
	return goFiles, instrumentedDir
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
