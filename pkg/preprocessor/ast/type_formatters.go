package ast

import (
	"go/ast"
)

func buildFormatCall(loggerPkg, funcName, paramName string) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent(loggerPkg),
			Sel: ast.NewIdent(funcName),
		},
		Args: []ast.Expr{ast.NewIdent(paramName)},
	}
}

func buildFormatArgWithLogger(paramName, paramType, loggerPkg string) ast.Expr {
	if len(paramType) > 6 && paramType[:6] == "slice:" {
		paramType = "slice"
	}

	switch paramType {
	case "bytes":
		return buildFormatCall(loggerPkg, "FormatBytes", paramName)
	case "int", "int8", "int16":
		return buildFormatCall(loggerPkg, "FormatInt", paramName)
	case "int32":
		return buildFormatCall(loggerPkg, "FormatInt32", paramName)
	case "int64":
		return buildFormatCall(loggerPkg, "FormatInt64", paramName)
	case "uint", "uint8", "uint16":
		return buildFormatCall(loggerPkg, "FormatUint", paramName)
	case "uint32":
		return buildFormatCall(loggerPkg, "FormatUint32", paramName)
	case "uint64", "uintptr":
		return buildFormatCall(loggerPkg, "FormatUint64", paramName)
	case "float32", "float64":
		return buildFormatCall(loggerPkg, "FormatFloat64", paramName)
	case "bool":
		return buildFormatCall(loggerPkg, "FormatBool", paramName)
	case "string":
		return buildFormatCall(loggerPkg, "FormatString", paramName)
	case "slice":
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(loggerPkg),
				Sel: ast.NewIdent("FormatInt"),
			},
			Args: []ast.Expr{
				&ast.CallExpr{
					Fun:  ast.NewIdent("len"),
					Args: []ast.Expr{ast.NewIdent(paramName)},
				},
			},
		}
	case "any", "interface":
		return buildFormatCall(loggerPkg, "FormatAny", paramName)
	case "Value":
		return buildFormatCall(loggerPkg, "FormatValue", paramName)
	default:
		return buildFormatCall(loggerPkg, "FormatAny", paramName)
	}
}

func buildFormatArg(paramName, paramType string) ast.Expr {
	return buildFormatArgWithLogger(paramName, paramType, "instrumentlog")
}
