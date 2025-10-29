package preprocessor

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
)

const instrumentationMarker = "// INSTRUMENTED BY GO-RUNTIME-OBSERVER"

func instrumentStdlibFile(filePath string, functions []string, methods []StdlibMethodInstrumentation) ([]byte, bool, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read file: %w", err)
	}

	// Check if already instrumented
	if bytes.Contains(content, []byte(instrumentationMarker)) {
		return content, false, nil
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return nil, false, fmt.Errorf("failed to parse file: %w", err)
	}

	modified := false

	if len(functions) > 0 {
		modified = instrumentFunctions(file, functions) || modified
	}

	if len(methods) > 0 {
		modified = instrumentMethods(file, methods) || modified
	}

	if !modified {
		return content, false, nil
	}

	addImportIfNeeded(file, "runtime_observe_instrumentation/instrumentlog")

	var buf []byte
	outputBuf := &bytesBuffer{buf: buf}
	if err := format.Node(outputBuf, fset, file); err != nil {
		return nil, false, fmt.Errorf("failed to format code: %w", err)
	}

	markerLine := []byte(instrumentationMarker + "\n")
	result := append(markerLine, outputBuf.buf...)

	return result, true, nil
}

type bytesBuffer struct {
	buf []byte
}

func (b *bytesBuffer) Write(p []byte) (n int, err error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func instrumentFunctions(file *ast.File, functionNames []string) bool {
	modified := false

	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Body == nil {
			continue
		}

		if funcDecl.Recv != nil {
			continue
		}

		funcName := funcDecl.Name.Name
		for _, targetName := range functionNames {
			if funcName == targetName {
				injectLogCall(funcDecl, funcName, "")
				modified = true
				break
			}
		}
	}

	return modified
}

func instrumentMethods(file *ast.File, methodConfigs []StdlibMethodInstrumentation) bool {
	modified := false

	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Body == nil {
			continue
		}

		if funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
			continue
		}

		receiverType := extractReceiverType(funcDecl.Recv.List[0].Type)
		methodName := funcDecl.Name.Name

		for _, methodConfig := range methodConfigs {
			if receiverType == methodConfig.ReceiverType {
				for _, targetMethod := range methodConfig.MethodNames {
					if methodName == targetMethod {
						injectLogCall(funcDecl, methodName, receiverType)
						modified = true
						break
					}
				}
			}
		}
	}

	return modified
}

func extractReceiverType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

func injectLogCall(funcDecl *ast.FuncDecl, name string, receiverType string) {
	var operationName string
	if receiverType != "" {
		operationName = fmt.Sprintf("%s.%s", receiverType, name)
	} else {
		operationName = name
	}

	logStmt := &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent("instrumentlog"),
				Sel: ast.NewIdent("LogCall"),
			},
			Args: []ast.Expr{
				&ast.BasicLit{
					Kind:  token.STRING,
					Value: fmt.Sprintf(`"reflect.%s"`, operationName),
				},
			},
		},
	}

	funcDecl.Body.List = append([]ast.Stmt{logStmt}, funcDecl.Body.List...)
}

func addImportIfNeeded(file *ast.File, importPath string) bool {
	for _, imp := range file.Imports {
		if imp.Path.Value == `"`+importPath+`"` {
			return false
		}
	}

	newImport := &ast.ImportSpec{
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: `"` + importPath + `"`,
		},
	}

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if ok && genDecl.Tok == token.IMPORT {
			genDecl.Specs = append(genDecl.Specs, newImport)
			return true
		}
	}

	newGenDecl := &ast.GenDecl{
		Tok:   token.IMPORT,
		Specs: []ast.Spec{newImport},
	}
	file.Decls = append([]ast.Decl{newGenDecl}, file.Decls...)

	return true
}
