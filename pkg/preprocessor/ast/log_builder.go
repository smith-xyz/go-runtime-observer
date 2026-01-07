package ast

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/smith-xyz/go-runtime-observer/pkg/preprocessor/types"
)

const (
	logCallFunctionName = "LogCall"
)

type logCallBuilder struct {
	packageName   string
	loggerPackage string
	operation     string
	args          map[string]ast.Expr
}

func newLogCallBuilder(packageName string, loggerType types.LoggerType) *logCallBuilder {
	loggerPackage := "instrumentlog"
	if loggerType == types.LoggerTypeFormat {
		loggerPackage = "formatlog"
	}
	return &logCallBuilder{
		packageName:   packageName,
		loggerPackage: loggerPackage,
		args:          make(map[string]ast.Expr),
	}
}

func (b *logCallBuilder) setOperation(name string, receiverType string) *logCallBuilder {
	if receiverType != "" {
		b.operation = fmt.Sprintf("%s.%s", receiverType, name)
	} else {
		b.operation = name
	}
	return b
}

func (b *logCallBuilder) addOperationArg() *logCallBuilder {
	return b
}

func (b *logCallBuilder) addParam(paramName string, paramType string) *logCallBuilder {
	b.args[paramName] = buildFormatArgWithLogger(paramName, paramType, b.loggerPackage)
	return b
}

func (b *logCallBuilder) addLiteralString(paramName string, value string) *logCallBuilder {
	b.args[paramName] = &ast.BasicLit{
		Kind:  token.STRING,
		Value: fmt.Sprintf(`"%s"`, value),
	}
	return b
}

func (b *logCallBuilder) build() ast.Stmt {
	operationLit := &ast.BasicLit{
		Kind:  token.STRING,
		Value: fmt.Sprintf(`"%s.%s"`, b.packageName, b.operation),
	}

	mapElements := make([]ast.Expr, 0, len(b.args))
	for key, value := range b.args {
		mapElements = append(mapElements,
			&ast.KeyValueExpr{
				Key: &ast.BasicLit{
					Kind:  token.STRING,
					Value: fmt.Sprintf(`"%s"`, key),
				},
				Value: value,
			},
		)
	}

	mapLit := &ast.CompositeLit{
		Type: &ast.SelectorExpr{
			X:   ast.NewIdent(b.loggerPackage),
			Sel: ast.NewIdent("CallArgs"),
		},
		Elts: mapElements,
	}

	return &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(b.loggerPackage),
				Sel: ast.NewIdent(logCallFunctionName),
			},
			Args: []ast.Expr{
				operationLit,
				mapLit,
			},
		},
	}
}
