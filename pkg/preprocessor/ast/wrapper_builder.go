package ast

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/smith-xyz/go-runtime-observer/pkg/preprocessor/types"
)

const instrumentedSuffix = "_instrumented"

type wrapperBuilder struct {
	file          *ast.File
	registry      types.Registry
	neededImports map[string]string
	modified      bool
}

func NewWrapperBuilder(file *ast.File, registry types.Registry) *wrapperBuilder {
	return &wrapperBuilder{
		file:          file,
		registry:      registry,
		neededImports: make(map[string]string),
		modified:      false,
	}
}

func (wb *wrapperBuilder) ReplaceInstrumentedCalls() *wrapperBuilder {
	ast.Inspect(wb.file, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok {
				packageName := ident.Name
				functionName := sel.Sel.Name

				if wb.registry.IsInstrumented(packageName, functionName) {
					aliasName := packageName + instrumentedSuffix
					sel.X = &ast.Ident{Name: aliasName}
					wb.neededImports[packageName] = aliasName
					wb.modified = true
				}
			}
		}
		return true
	})
	return wb
}

func (wb *wrapperBuilder) AddInstrumentedImports() *wrapperBuilder {
	for packageName, aliasName := range wb.neededImports {
		if instrumentedPath, exists := wb.registry.GetInstrumentedImportPath(packageName); exists {
			ast.Inspect(wb.file, func(n ast.Node) bool {
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
						wb.modified = true
					}
				}
				return true
			})
		}
	}
	return wb
}

func (wb *wrapperBuilder) RemoveUnusedImports() *wrapperBuilder {
	for packageName := range wb.neededImports {
		hasNonInstrumentedUses := false
		ast.Inspect(wb.file, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok {
					if ident.Name == packageName {
						functionName := sel.Sel.Name
						if !wb.registry.IsInstrumented(packageName, functionName) {
							hasNonInstrumentedUses = true
							return false
						}
					}
				}
			}
			return true
		})

		if !hasNonInstrumentedUses {
			ast.Inspect(wb.file, func(n ast.Node) bool {
				if decl, ok := n.(*ast.GenDecl); ok && decl.Tok == token.IMPORT {
					filteredSpecs := []ast.Spec{}
					for _, spec := range decl.Specs {
						if impSpec, ok := spec.(*ast.ImportSpec); ok && impSpec.Path != nil {
							importPath := impSpec.Path.Value
							if strings.HasPrefix(importPath, `"`) && strings.HasSuffix(importPath, `"`) {
								unquotedPath := importPath[1 : len(importPath)-1]
								if unquotedPath == packageName {
									wb.modified = true
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
	return wb
}

func (wb *wrapperBuilder) IsModified() bool {
	return wb.modified
}
