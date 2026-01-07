package ast

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/token"

	"github.com/smith-xyz/go-runtime-observer/pkg/preprocessor/types"
)

const instrumentationMarker = "// INSTRUMENTED BY GO-RUNTIME-OBSERVER"

type fileInjector struct {
	file        *ast.File
	fset        *token.FileSet
	packageName string
	loggerType  types.LoggerType
	modified    bool
}

func NewFileInjector(file *ast.File, fset *token.FileSet, packageName string) *fileInjector {
	return &fileInjector{
		file:        file,
		fset:        fset,
		packageName: packageName,
		loggerType:  types.LoggerTypeInstrument,
		modified:    false,
	}
}

func NewFileInjectorWithLogger(file *ast.File, fset *token.FileSet, packageName string, loggerType types.LoggerType) *fileInjector {
	return &fileInjector{
		file:        file,
		fset:        fset,
		packageName: packageName,
		loggerType:  loggerType,
		modified:    false,
	}
}

func (fi *fileInjector) InjectFunctions(functionNames []string) bool {
	for _, decl := range fi.file.Decls {
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
				fi.injectLogCall(funcDecl, funcName, "", "", false)
				fi.modified = true
				break
			}
		}
	}

	return fi.modified
}

func (fi *fileInjector) InjectMethods(methodConfigs []types.StdlibMethodInstrumentation) bool {
	for _, decl := range fi.file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Body == nil {
			continue
		}

		if funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
			continue
		}

		receiverType := extractReceiverType(funcDecl.Recv.List[0].Type)
		methodName := funcDecl.Name.Name
		receiverName := extractReceiverName(funcDecl.Recv.List[0])

		for _, methodConfig := range methodConfigs {
			if receiverType == methodConfig.ReceiverType {
				for _, targetMethod := range methodConfig.MethodNames {
					if methodName == targetMethod {
						shouldLookup := false
						for _, lookupMethod := range methodConfig.CorrelationLookupMethods {
							if methodName == lookupMethod {
								shouldLookup = true
								break
							}
						}
						fi.injectLogCall(funcDecl, methodName, receiverType, receiverName, shouldLookup)

						for _, correlationMethod := range methodConfig.CorrelationRecordingMethods {
							if methodName == correlationMethod {
								extractor := ""
								if methodConfig.MethodIdentifierExtractors != nil {
									extractor = methodConfig.MethodIdentifierExtractors[methodName]
								}
								returnMethods := []string{}
								if methodConfig.ReturnExpressionMethods != nil {
									if methods, ok := methodConfig.ReturnExpressionMethods[methodName]; ok {
										returnMethods = methods
									}
								}
								fi.injectCorrelationRecording(funcDecl, methodName, receiverName, extractor, returnMethods)
								break
							}
						}

						fi.modified = true
						break
					}
				}
			}
		}
	}

	return fi.modified
}

func (fi *fileInjector) AddImport(importPath string) bool {
	for _, imp := range fi.file.Imports {
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

	for _, decl := range fi.file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if ok && genDecl.Tok == token.IMPORT {
			genDecl.Specs = append(genDecl.Specs, newImport)
			fi.modified = true
			return true
		}
	}

	newGenDecl := &ast.GenDecl{
		Tok:   token.IMPORT,
		Specs: []ast.Spec{newImport},
	}
	fi.file.Decls = append([]ast.Decl{newGenDecl}, fi.file.Decls...)
	fi.modified = true

	return true
}

func (fi *fileInjector) GetLoggerImportPath() string {
	if fi.loggerType == types.LoggerTypeFormat {
		return "runtime_observe_instrumentation/formatlog"
	}
	return "runtime_observe_instrumentation/instrumentlog"
}

func (fi *fileInjector) GetLoggerPackageName() string {
	if fi.loggerType == types.LoggerTypeFormat {
		return "formatlog"
	}
	return "instrumentlog"
}

func (fi *fileInjector) Render() ([]byte, error) {
	var buf []byte
	outputBuf := &bytesBuffer{buf: buf}
	if err := format.Node(outputBuf, fi.fset, fi.file); err != nil {
		return nil, fmt.Errorf("failed to format code: %w", err)
	}

	markerLine := []byte(instrumentationMarker + "\n")
	result := append(markerLine, outputBuf.buf...)

	return result, nil
}

func (fi *fileInjector) injectLogCall(funcDecl *ast.FuncDecl, name string, receiverType string, receiverName string, shouldLookup bool) {
	builder := newLogCallBuilder(fi.packageName, fi.loggerType).
		setOperation(name, receiverType).
		addOperationArg()

	if receiverName != "" && funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		receiverTypeStr := getTypeString(funcDecl.Recv.List[0].Type)
		builder.addParam(receiverName, receiverTypeStr)
	}

	if funcDecl.Type.Params != nil {
		for _, field := range funcDecl.Type.Params.List {
			if len(field.Names) == 0 {
				continue
			}
			for _, paramName := range field.Names {
				if paramName.Name == "" {
					continue
				}

				paramType := getTypeString(field.Type)
				builder.addParam(paramName.Name, paramType)
			}
		}
	}

	if shouldLookup {
		builder.addLiteralString("_correlation_lookup", "true")
	}

	logStmt := builder.build()
	funcDecl.Body.List = append([]ast.Stmt{logStmt}, funcDecl.Body.List...)
}

func (fi *fileInjector) injectCorrelationRecording(funcDecl *ast.FuncDecl, methodName string, receiverName string, extractorSpec string, returnMethods []string) {
	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		retStmt, ok := n.(*ast.ReturnStmt)
		if !ok || len(retStmt.Results) == 0 {
			return true
		}

		returnExpr := retStmt.Results[0]
		var identifierExpr ast.Expr

		if callExpr, ok := returnExpr.(*ast.CallExpr); ok {
			if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
				shouldProcess := false
				if len(returnMethods) == 0 {
					shouldProcess = selExpr.Sel.Name == methodName
				} else {
					for _, returnMethod := range returnMethods {
						if selExpr.Sel.Name == returnMethod {
							shouldProcess = true
							break
						}
					}
					if !shouldProcess && selExpr.Sel.Name == methodName {
						shouldProcess = true
					}
				}

				if shouldProcess {
					identifierExpr = extractIdentifierExpr(funcDecl, callExpr, extractorSpec)

					if identifierExpr != nil {
						recordStmt := fi.buildRecordMethodByNameCall(returnExpr, identifierExpr, ast.NewIdent(receiverName))
						fi.insertBeforeReturn(retStmt, recordStmt)
						fi.modified = true
					}
				}
			}
		} else {
			if len(returnMethods) == 0 {
				identifierExpr = extractIdentifierExpr(funcDecl, nil, extractorSpec)
				if identifierExpr != nil {
					recordStmt := fi.buildRecordMethodByNameCall(returnExpr, identifierExpr, ast.NewIdent(receiverName))
					fi.insertBeforeReturn(retStmt, recordStmt)
					fi.modified = true
				}
			}
		}

		return true
	})
}

func extractIdentifierExpr(funcDecl *ast.FuncDecl, callExpr *ast.CallExpr, extractorSpec string) ast.Expr {
	if extractorSpec == "" {
		return nil
	}

	if len(extractorSpec) > 6 && extractorSpec[:6] == "param:" {
		paramName := extractorSpec[6:]
		return extractParameterExpr(funcDecl, paramName)
	} else if len(extractorSpec) > 5 && extractorSpec[:5] == "call:" {
		if callExpr == nil {
			return nil
		}
		argIndex := 0
		if len(extractorSpec) > 5 {
			if idx := parseArgIndex(extractorSpec[5:]); idx >= 0 {
				argIndex = idx
			}
		}
		if argIndex < len(callExpr.Args) {
			return callExpr.Args[argIndex]
		}
	}

	return nil
}

func extractParameterExpr(funcDecl *ast.FuncDecl, paramName string) ast.Expr {
	if funcDecl.Type.Params == nil {
		return nil
	}
	for _, field := range funcDecl.Type.Params.List {
		for _, name := range field.Names {
			if name.Name == paramName {
				return ast.NewIdent(name.Name)
			}
		}
	}
	return nil
}

func parseArgIndex(s string) int {
	if len(s) == 0 {
		return 0
	}
	result := 0
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			result = result*10 + int(s[i]-'0')
		} else {
			break
		}
	}
	return result
}

func (fi *fileInjector) buildRecordMethodByNameCall(methodValueExpr ast.Expr, nameParamExpr ast.Expr, receiverNameExpr ast.Expr) ast.Stmt {
	return &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(fi.GetLoggerPackageName()),
				Sel: ast.NewIdent("RecordMethodByName"),
			},
			Args: []ast.Expr{
				methodValueExpr,
				nameParamExpr,
				receiverNameExpr,
			},
		},
	}
}

func (fi *fileInjector) insertBeforeReturn(retStmt *ast.ReturnStmt, logStmt ast.Stmt) {
	found := false
	ast.Inspect(fi.file, func(n ast.Node) bool {
		if found {
			return false
		}
		block, ok := n.(*ast.BlockStmt)
		if !ok {
			return true
		}

		for i, stmt := range block.List {
			if stmt == retStmt {
				newList := make([]ast.Stmt, 0, len(block.List)+1)
				newList = append(newList, block.List[:i]...)
				newList = append(newList, logStmt)
				newList = append(newList, block.List[i:]...)
				block.List = newList
				found = true
				return false
			}
		}
		return true
	})
}

func (fi *fileInjector) IsModified() bool {
	return fi.modified
}

type bytesBuffer struct {
	buf []byte
}

func (b *bytesBuffer) Write(p []byte) (n int, err error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}
