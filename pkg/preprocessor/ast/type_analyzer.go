package ast

import "go/ast"

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

func extractReceiverName(field *ast.Field) string {
	if len(field.Names) > 0 && field.Names[0].Name != "" {
		return field.Names[0].Name
	}
	return ""
}

func getTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.ArrayType:
		elemType := getTypeString(t.Elt)
		if elemType == "byte" {
			return "bytes"
		}
		if elemType != "" {
			return "slice:" + elemType
		}
		return "slice"
	case *ast.SelectorExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name + "." + t.Sel.Name
		}
	case *ast.InterfaceType:
		return "interface"
	case *ast.Ellipsis:
		elemType := getTypeString(t.Elt)
		if elemType == "byte" {
			return "bytes"
		}
		if elemType != "" {
			return "slice:" + elemType
		}
		return "slice"
	case *ast.StarExpr:
		elemType := getTypeString(t.X)
		if elemType != "" {
			return "*" + elemType
		}
		return "pointer"
	}
	return "unknown"
}
