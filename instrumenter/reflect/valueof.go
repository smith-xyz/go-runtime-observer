package reflect

import (
	"go/ast"
	"go/token"
)

func InjectValueOf(fn *ast.FuncDecl) {
	logStmt := &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent("instrumentlog"),
				Sel: ast.NewIdent("LogCall"),
			},
			Args: []ast.Expr{
				&ast.BasicLit{
					Kind:  token.STRING,
					Value: `"ValueOf"`,
				},
			},
		},
	}

	fn.Body.List = append([]ast.Stmt{logStmt}, fn.Body.List...)
}

