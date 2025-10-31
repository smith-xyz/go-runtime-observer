package ast

import (
	"go/ast"
)

const instrumentlogPackageName = "instrumentlog"

type typeFormatter func(paramName string) ast.Expr

var typeFormatters = map[string]typeFormatter{
	"bytes": func(paramName string) ast.Expr {
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(instrumentlogPackageName),
				Sel: ast.NewIdent("FormatBytes"),
			},
			Args: []ast.Expr{ast.NewIdent(paramName)},
		}
	},
	"int": func(paramName string) ast.Expr {
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(instrumentlogPackageName),
				Sel: ast.NewIdent("FormatInt"),
			},
			Args: []ast.Expr{ast.NewIdent(paramName)},
		}
	},
	"int8": func(paramName string) ast.Expr {
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(instrumentlogPackageName),
				Sel: ast.NewIdent("FormatInt"),
			},
			Args: []ast.Expr{ast.NewIdent(paramName)},
		}
	},
	"int16": func(paramName string) ast.Expr {
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(instrumentlogPackageName),
				Sel: ast.NewIdent("FormatInt"),
			},
			Args: []ast.Expr{ast.NewIdent(paramName)},
		}
	},
	"int32": func(paramName string) ast.Expr {
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(instrumentlogPackageName),
				Sel: ast.NewIdent("FormatInt"),
			},
			Args: []ast.Expr{ast.NewIdent(paramName)},
		}
	},
	"int64": func(paramName string) ast.Expr {
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(instrumentlogPackageName),
				Sel: ast.NewIdent("FormatInt64"),
			},
			Args: []ast.Expr{ast.NewIdent(paramName)},
		}
	},
	"uint": func(paramName string) ast.Expr {
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(instrumentlogPackageName),
				Sel: ast.NewIdent("FormatUint"),
			},
			Args: []ast.Expr{ast.NewIdent(paramName)},
		}
	},
	"uint8": func(paramName string) ast.Expr {
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(instrumentlogPackageName),
				Sel: ast.NewIdent("FormatUint"),
			},
			Args: []ast.Expr{ast.NewIdent(paramName)},
		}
	},
	"uint16": func(paramName string) ast.Expr {
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(instrumentlogPackageName),
				Sel: ast.NewIdent("FormatUint"),
			},
			Args: []ast.Expr{ast.NewIdent(paramName)},
		}
	},
	"uint32": func(paramName string) ast.Expr {
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(instrumentlogPackageName),
				Sel: ast.NewIdent("FormatUint"),
			},
			Args: []ast.Expr{ast.NewIdent(paramName)},
		}
	},
	"uint64": func(paramName string) ast.Expr {
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(instrumentlogPackageName),
				Sel: ast.NewIdent("FormatUint64"),
			},
			Args: []ast.Expr{ast.NewIdent(paramName)},
		}
	},
	"uintptr": func(paramName string) ast.Expr {
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(instrumentlogPackageName),
				Sel: ast.NewIdent("FormatUint64"),
			},
			Args: []ast.Expr{ast.NewIdent(paramName)},
		}
	},
	"float32": func(paramName string) ast.Expr {
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(instrumentlogPackageName),
				Sel: ast.NewIdent("FormatFloat64"),
			},
			Args: []ast.Expr{ast.NewIdent(paramName)},
		}
	},
	"float64": func(paramName string) ast.Expr {
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(instrumentlogPackageName),
				Sel: ast.NewIdent("FormatFloat64"),
			},
			Args: []ast.Expr{ast.NewIdent(paramName)},
		}
	},
	"bool": func(paramName string) ast.Expr {
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(instrumentlogPackageName),
				Sel: ast.NewIdent("FormatBool"),
			},
			Args: []ast.Expr{ast.NewIdent(paramName)},
		}
	},
	"string": func(paramName string) ast.Expr {
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(instrumentlogPackageName),
				Sel: ast.NewIdent("FormatString"),
			},
			Args: []ast.Expr{ast.NewIdent(paramName)},
		}
	},
	"slice": func(paramName string) ast.Expr {
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(instrumentlogPackageName),
				Sel: ast.NewIdent("FormatInt"),
			},
			Args: []ast.Expr{
				&ast.CallExpr{
					Fun:  ast.NewIdent("len"),
					Args: []ast.Expr{ast.NewIdent(paramName)},
				},
			},
		}
	},
	"any": func(paramName string) ast.Expr {
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(instrumentlogPackageName),
				Sel: ast.NewIdent("FormatAny"),
			},
			Args: []ast.Expr{ast.NewIdent(paramName)},
		}
	},
	"interface": func(paramName string) ast.Expr {
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(instrumentlogPackageName),
				Sel: ast.NewIdent("FormatAny"),
			},
			Args: []ast.Expr{ast.NewIdent(paramName)},
		}
	},
}

func buildFormatArg(paramName, paramType string) ast.Expr {
	if len(paramType) > 6 && paramType[:6] == "slice:" {
		paramType = "slice"
	}

	if formatter, ok := typeFormatters[paramType]; ok {
		return formatter(paramName)
	}

	// For reflect.Value type, use FormatValue to extract internal ptr
	// This allows matching return values with Call receivers
	if paramType == "Value" {
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(instrumentlogPackageName),
				Sel: ast.NewIdent("FormatValue"),
			},
			Args: []ast.Expr{ast.NewIdent(paramName)},
		}
	}

	// For unknown types (like reflect.Type), use FormatAny
	// which will use FormatPointer to get unique instance addresses
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent(instrumentlogPackageName),
			Sel: ast.NewIdent("FormatAny"),
		},
		Args: []ast.Expr{ast.NewIdent(paramName)},
	}
}
