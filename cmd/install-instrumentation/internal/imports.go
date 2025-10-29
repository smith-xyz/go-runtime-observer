package internal

import (
	"fmt"
	"go/ast"
	"go/token"
)

const (
	IMPORT_VALUE           = `"runtime_observe_instrumentation/preprocessor"`
	IMPORT_BLOCK_NOT_FOUND = "no import block found"
)

func AddImport(file *ast.File) error {
	for _, imp := range file.Imports {
		if imp.Path != nil && imp.Path.Value == IMPORT_VALUE {
			return nil
		}
	}

	var importDecl *ast.GenDecl
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			importDecl = genDecl
			break
		}
	}

	if importDecl == nil {
		return fmt.Errorf("%s", IMPORT_BLOCK_NOT_FOUND)
	}

	newImport := &ast.ImportSpec{
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: IMPORT_VALUE,
		},
	}

	importDecl.Specs = append(importDecl.Specs, newImport)
	return nil
}

func HasImport(file *ast.File) bool {
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			for _, spec := range genDecl.Specs {
				if impSpec, ok := spec.(*ast.ImportSpec); ok && impSpec.Path != nil && impSpec.Path.Value == IMPORT_VALUE {
					return true
				}
			}
		}
	}
	return false
}
