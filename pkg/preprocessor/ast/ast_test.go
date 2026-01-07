package ast

import (
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/smith-xyz/go-runtime-observer/pkg/preprocessor/types"
)

func TestNewFileInjector(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", "package test", parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	injector := NewFileInjector(file, fset, "testpkg")
	if injector == nil {
		t.Fatal("NewFileInjector returned nil")
	}

	if injector.IsModified() {
		t.Error("New injector should not be modified")
	}
}

func TestFileInjector_InjectFunctions(t *testing.T) {
	src := `package reflect

func ValueOf(i any) Value {
	return Value{}
}

func TypeOf(i interface{}) Type {
	return Type{}
}

func OtherFunc() {
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	injector := NewFileInjector(file, fset, "reflect")

	modified := injector.InjectFunctions([]string{"ValueOf", "TypeOf"})
	if !modified {
		t.Error("Expected injector to be modified")
	}

	result, err := injector.Render()
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := string(result)
	if !strings.Contains(output, `instrumentlog.LogCall("reflect.ValueOf"`) {
		t.Error("Expected ValueOf to be instrumented")
	}
	if !strings.Contains(output, `instrumentlog.LogCall("reflect.TypeOf"`) {
		t.Error("Expected TypeOf to be instrumented")
	}
	if strings.Contains(output, `instrumentlog.LogCall("reflect.OtherFunc"`) {
		t.Error("OtherFunc should not be instrumented")
	}
}

func TestFileInjector_InjectMethods(t *testing.T) {
	src := `package reflect

type Value struct{}

func (v Value) Call(args []Value) []Value {
	return nil
}

func (v Value) Set(x Value) {
}

func (v Value) OtherMethod() {
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	injector := NewFileInjector(file, fset, "reflect")

	modified := injector.InjectMethods([]types.StdlibMethodInstrumentation{
		{
			ReceiverType: "Value",
			MethodNames:  []string{"Call", "Set"},
		},
	})
	if !modified {
		t.Error("Expected injector to be modified")
	}

	result, err := injector.Render()
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := string(result)
	if !strings.Contains(output, `instrumentlog.LogCall("reflect.Value.Call"`) {
		t.Error("Expected Call method to be instrumented")
	}
	if !strings.Contains(output, `"v"`) {
		t.Error("Expected receiver 'v' to be logged for Call method")
	}
	if !strings.Contains(output, `instrumentlog.LogCall("reflect.Value.Set"`) {
		t.Error("Expected Set method to be instrumented")
	}
	if strings.Contains(output, `instrumentlog.LogCall("reflect.Value.OtherMethod"`) {
		t.Error("OtherMethod should not be instrumented")
	}
}

func TestFileInjector_InjectMethods_WithCorrelationRecording(t *testing.T) {
	src := `package reflect

type Value struct{}

func (v Value) MethodByName(name string) Value {
	return v.Method(0)
}

func (v Value) Method(i int) Value {
	return Value{}
}

func (v Value) Call(args []Value) []Value {
	return nil
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	injector := NewFileInjector(file, fset, "reflect")

	modified := injector.InjectMethods([]types.StdlibMethodInstrumentation{
		{
			ReceiverType:                "Value",
			MethodNames:                 []string{"MethodByName", "Method", "Call"},
			CorrelationRecordingMethods: []string{"MethodByName", "Method"},
			MethodIdentifierExtractors: map[string]string{
				"MethodByName": "param:name",
				"Method":       "call:0",
			},
			ReturnExpressionMethods: map[string][]string{
				"MethodByName": {"Method"},
				"Method":       {},
			},
		},
	})
	if !modified {
		t.Error("Expected injector to be modified")
	}

	result, err := injector.Render()
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := string(result)

	// Verify MethodByName has correlation recording
	if !strings.Contains(output, `instrumentlog.RecordMethodByName`) {
		t.Error("Expected MethodByName to have correlation recording")
	}
	if !strings.Contains(output, `instrumentlog.LogCall("reflect.Value.MethodByName"`) {
		t.Error("Expected MethodByName to have log call")
	}

	// Verify Method has correlation recording
	if !strings.Contains(output, `instrumentlog.RecordMethodByName`) {
		t.Error("Expected Method to have correlation recording")
	}

	// Verify Call does NOT have correlation recording (only MethodByName and Method do)
	callIndex := strings.Index(output, `func (v Value) Call`)
	if callIndex > 0 {
		callSection := output[callIndex:]
		if strings.Contains(callSection, `instrumentlog.RecordMethodByName`) {
			t.Error("Call should not have correlation recording")
		}
	}
}

func TestFileInjector_InjectMethods_MultipleReturnStatements(t *testing.T) {
	src := `package reflect

type Value struct{}

func (v Value) MethodByName(name string) Value {
	if name == "" {
		return v.Method(0)
	}
	return v.Method(1)
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	injector := NewFileInjector(file, fset, "reflect")

	modified := injector.InjectMethods([]types.StdlibMethodInstrumentation{
		{
			ReceiverType:                "Value",
			MethodNames:                 []string{"MethodByName"},
			CorrelationRecordingMethods: []string{"MethodByName"},
			MethodIdentifierExtractors: map[string]string{
				"MethodByName": "param:name",
			},
			ReturnExpressionMethods: map[string][]string{
				"MethodByName": {"Method"},
			},
		},
	})
	if !modified {
		t.Error("Expected injector to be modified")
	}

	result, err := injector.Render()
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := string(result)

	// Verify correlation recording is injected for return statements that call Method
	recordCount := strings.Count(output, `instrumentlog.RecordMethodByName`)
	if recordCount < 2 {
		t.Errorf("Expected at least 2 correlation recording calls, got %d", recordCount)
	}
}

func TestFileInjector_AddImport(t *testing.T) {
	src := `package test

import "fmt"

func main() {
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	injector := NewFileInjector(file, fset, "test")

	added := injector.AddImport("runtime_observe_instrumentation/instrumentlog")
	if !added {
		t.Error("Expected import to be added")
	}

	if !injector.IsModified() {
		t.Error("Expected injector to be modified")
	}

	result, err := injector.Render()
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := string(result)
	if !strings.Contains(output, `"runtime_observe_instrumentation/instrumentlog"`) {
		t.Error("Expected instrumentlog import to be added")
	}

	// Re-parse to check duplicate import detection
	file2, err := parser.ParseFile(fset, "test.go", output, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to re-parse: %v", err)
	}

	injector2 := NewFileInjector(file2, fset, "test")
	addedAgain := injector2.AddImport("runtime_observe_instrumentation/instrumentlog")
	if addedAgain {
		t.Error("Import should not be added again")
	}
}

func TestFileInjector_InjectFunctionsWithParams(t *testing.T) {
	src := `package reflect

func ValueOf(i any) Value {
	return Value{}
}

func MakeSlice(typ Type, len, cap int) Value {
	return Value{}
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	injector := NewFileInjector(file, fset, "reflect")
	injector.InjectFunctions([]string{"ValueOf", "MakeSlice"})

	result, err := injector.Render()
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := string(result)
	if !strings.Contains(output, `"i"`) {
		t.Error("Expected parameter 'i' to be logged")
	}
	if !strings.Contains(output, `"len"`) || !strings.Contains(output, `"cap"`) {
		t.Error("Expected parameters 'len' and 'cap' to be logged")
	}
}

func TestFileInjector_InjectFunctionsWithAnyParam(t *testing.T) {
	src := `package reflect

func ValueOf(i any) Value {
	return Value{}
}

func TypeOf(i interface{}) Type {
	return Type{}
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	injector := NewFileInjector(file, fset, "reflect")
	injector.InjectFunctions([]string{"ValueOf", "TypeOf"})

	result, err := injector.Render()
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := string(result)

	// Verify that 'any' parameter uses FormatAny
	if !strings.Contains(output, `instrumentlog.FormatAny(i)`) {
		t.Errorf("Expected 'any' parameter to use FormatAny. Output:\n%s", output)
	}

	// Verify that 'interface{}' parameter also uses FormatAny
	// Both ValueOf and TypeOf have parameter 'i', so we should see FormatAny twice
	count := strings.Count(output, `instrumentlog.FormatAny(i)`)
	if count < 2 {
		t.Errorf("Expected FormatAny to be called at least twice (for both any and interface{}), got %d. Output:\n%s", count, output)
	}

	// Verify the LogCall includes the parameter
	if !strings.Contains(output, `"i"`) {
		t.Error("Expected parameter 'i' to be logged")
	}
}

func TestNewWrapperBuilder(t *testing.T) {
	src := `package main

import "unsafe"

func main() {
	_ = unsafe.Add(nil, 8)
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	mockRegistry := &mockRegistry{
		instrumented: map[string]map[string]bool{
			"unsafe": {"Add": true},
		},
		paths: map[string]string{
			"unsafe": "runtime_observe_instrumentation/unsafe",
		},
	}

	builder := NewWrapperBuilder(file, mockRegistry)
	if builder == nil {
		t.Fatal("NewWrapperBuilder returned nil")
	}

	if builder.IsModified() {
		t.Error("New builder should not be modified")
	}
}

func TestWrapperBuilder_ReplaceInstrumentedCalls(t *testing.T) {
	src := `package main

import "unsafe"

func main() {
	ptr := unsafe.Pointer(nil)
	_ = unsafe.Add(ptr, 8)
	_ = unsafe.Slice(nil, 8)
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	mockRegistry := &mockRegistry{
		instrumented: map[string]map[string]bool{
			"unsafe": {"Add": true, "Slice": true},
		},
		paths: map[string]string{
			"unsafe": "runtime_observe_instrumentation/unsafe",
		},
	}

	builder := NewWrapperBuilder(file, mockRegistry)
	builder.ReplaceInstrumentedCalls()

	if !builder.IsModified() {
		t.Error("Expected builder to be modified")
	}

	var buf strings.Builder
	if err := format.Node(&buf, fset, file); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "unsafe_instrumented.Add") {
		t.Error("Expected unsafe.Add to be replaced with unsafe_instrumented.Add")
	}
	if !strings.Contains(output, "unsafe_instrumented.Slice") {
		t.Error("Expected unsafe.Slice to be replaced with unsafe_instrumented.Slice")
	}
	if !strings.Contains(output, "unsafe.Pointer") {
		t.Error("Expected unsafe.Pointer to remain unchanged")
	}
}

func TestWrapperBuilder_AddInstrumentedImports(t *testing.T) {
	src := `package main

import "unsafe"

func main() {
	_ = unsafe.Add(nil, 8)
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	mockRegistry := &mockRegistry{
		instrumented: map[string]map[string]bool{
			"unsafe": {"Add": true},
		},
		paths: map[string]string{
			"unsafe": "runtime_observe_instrumentation/unsafe",
		},
	}

	builder := NewWrapperBuilder(file, mockRegistry)
	builder.ReplaceInstrumentedCalls()
	builder.AddInstrumentedImports()

	var buf strings.Builder
	if err := format.Node(&buf, fset, file); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `unsafe_instrumented "runtime_observe_instrumentation/unsafe"`) {
		t.Error("Expected instrumented import to be added")
	}
}

func TestWrapperBuilder_RemoveUnusedImports(t *testing.T) {
	src := `package main

import "unsafe"

func main() {
	_ = unsafe.Add(nil, 8)
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	mockRegistry := &mockRegistry{
		instrumented: map[string]map[string]bool{
			"unsafe": {"Add": true},
		},
		paths: map[string]string{
			"unsafe": "runtime_observe_instrumentation/unsafe",
		},
	}

	builder := NewWrapperBuilder(file, mockRegistry)
	builder.ReplaceInstrumentedCalls()
	builder.AddInstrumentedImports()
	builder.RemoveUnusedImports()

	var buf strings.Builder
	if err := format.Node(&buf, fset, file); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, `import "unsafe"`) {
		t.Error("Expected original unsafe import to be removed")
	}
	if !strings.Contains(output, `unsafe_instrumented "runtime_observe_instrumentation/unsafe"`) {
		t.Error("Expected instrumented import to remain")
	}
}

func TestWrapperBuilder_KeepOriginalImportWhenMixed(t *testing.T) {
	src := `package main

import "unsafe"

func main() {
	_ = unsafe.Add(nil, 8)
	_ = unsafe.Pointer(nil)
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	mockRegistry := &mockRegistry{
		instrumented: map[string]map[string]bool{
			"unsafe": {"Add": true},
		},
		paths: map[string]string{
			"unsafe": "runtime_observe_instrumentation/unsafe",
		},
	}

	builder := NewWrapperBuilder(file, mockRegistry)
	builder.ReplaceInstrumentedCalls()
	builder.AddInstrumentedImports()
	builder.RemoveUnusedImports()

	var buf strings.Builder
	if err := format.Node(&buf, fset, file); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// After ReplaceInstrumentedCalls, unsafe.Add becomes unsafe_instrumented.Add
	// but unsafe.Pointer should remain as unsafe.Pointer
	// So we should have both the original import (for Pointer) and the instrumented import (for Add)
	hasOriginalImport := strings.Contains(output, `import "unsafe"`) || strings.Contains(output, `import (`) && strings.Contains(output, `"unsafe"`)
	hasInstrumentedImport := strings.Contains(output, `unsafe_instrumented "runtime_observe_instrumentation/unsafe"`) ||
		strings.Contains(output, `"runtime_observe_instrumentation/unsafe"`)

	if !hasOriginalImport {
		t.Errorf("Expected original unsafe import to remain (mixed usage). Output:\n%s", output)
	}
	if !hasInstrumentedImport {
		t.Error("Expected instrumented import to be added")
	}
}

func TestLogCallBuilder_WithParams(t *testing.T) {
	builder := newLogCallBuilder("reflect", types.LoggerTypeInstrument)
	builder.setOperation("ValueOf", "")
	builder.addOperationArg()
	builder.addParam("i", "any")
	builder.addParam("x", "int")
	builder.addParam("y", "string")

	stmt := builder.build()

	callExpr, ok := stmt.(*ast.ExprStmt).X.(*ast.CallExpr)
	if !ok {
		t.Fatal("Expected CallExpr")
	}

	selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		t.Fatal("Expected SelectorExpr")
	}

	if selExpr.X.(*ast.Ident).Name != "instrumentlog" {
		t.Errorf("Expected package name instrumentlog, got %s", selExpr.X.(*ast.Ident).Name)
	}

	if selExpr.Sel.Name != logCallFunctionName {
		t.Errorf("Expected function name %s, got %s", logCallFunctionName, selExpr.Sel.Name)
	}

	if len(callExpr.Args) != 2 {
		t.Errorf("Expected 2 args (operation + map), got %d", len(callExpr.Args))
	}

	opArg, ok := callExpr.Args[0].(*ast.BasicLit)
	if !ok {
		t.Fatal("Expected BasicLit for operation")
	}
	if !strings.Contains(opArg.Value, "reflect.ValueOf") {
		t.Errorf("Expected operation to contain 'reflect.ValueOf', got %s", opArg.Value)
	}

	mapLit, ok := callExpr.Args[1].(*ast.CompositeLit)
	if !ok {
		t.Fatal("Expected CompositeLit for map")
	}

	mapType, ok := mapLit.Type.(*ast.SelectorExpr)
	if !ok {
		t.Fatal("Expected SelectorExpr for map type")
	}
	if mapType.X.(*ast.Ident).Name != "instrumentlog" {
		t.Errorf("Expected package name instrumentlog, got %s", mapType.X.(*ast.Ident).Name)
	}
	if mapType.Sel.Name != "CallArgs" {
		t.Errorf("Expected CallArgs type, got %s", mapType.Sel.Name)
	}

	// Verify map has 3 key-value pairs (i, x, y)
	if len(mapLit.Elts) != 3 {
		t.Errorf("Expected 3 elements (3 key-value pairs), got %d", len(mapLit.Elts))
	}

	// Verify elements are KeyValueExpr nodes
	for _, elt := range mapLit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			t.Errorf("Expected KeyValueExpr, got %T", elt)
			continue
		}
		key, ok := kv.Key.(*ast.BasicLit)
		if !ok {
			t.Errorf("Expected BasicLit for key, got %T", kv.Key)
		}
		_ = key // Check that key is a string literal
	}
}

func TestLogCallBuilder_WithReceiver(t *testing.T) {
	builder := newLogCallBuilder("reflect", types.LoggerTypeInstrument)
	builder.setOperation("Call", "Value")
	builder.addOperationArg()

	stmt := builder.build()
	callExpr := stmt.(*ast.ExprStmt).X.(*ast.CallExpr)

	opArg, ok := callExpr.Args[0].(*ast.BasicLit)
	if !ok {
		t.Fatal("Expected BasicLit for operation")
	}

	if !strings.Contains(opArg.Value, "reflect.Value.Call") {
		t.Errorf("Expected operation to contain 'reflect.Value.Call', got %s", opArg.Value)
	}
}

func TestGetTypeString(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		expected string
	}{
		{
			name:     "int",
			src:      "package test; func f(x int) {}",
			expected: "int",
		},
		{
			name:     "string",
			src:      "package test; func f(x string) {}",
			expected: "string",
		},
		{
			name:     "[]byte",
			src:      "package test; func f(x []byte) {}",
			expected: "bytes",
		},
		{
			name:     "[]int",
			src:      "package test; func f(x []int) {}",
			expected: "slice:int",
		},
		{
			name:     "*int",
			src:      "package test; func f(x *int) {}",
			expected: "*int",
		},
		{
			name:     "interface{}",
			src:      "package test; func f(x interface{}) {}",
			expected: "interface",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.src, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			var funcDecl *ast.FuncDecl
			for _, decl := range file.Decls {
				if fd, ok := decl.(*ast.FuncDecl); ok {
					funcDecl = fd
					break
				}
			}

			if funcDecl == nil || funcDecl.Type.Params == nil || len(funcDecl.Type.Params.List) == 0 {
				t.Fatal("Failed to find function parameter")
			}

			paramType := getTypeString(funcDecl.Type.Params.List[0].Type)
			if paramType != tt.expected {
				t.Errorf("getTypeString() = %q, want %q", paramType, tt.expected)
			}
		})
	}
}

func TestExtractReceiverType(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		expected string
	}{
		{
			name:     "Value receiver",
			src:      "package test; func (v Value) Call() {}",
			expected: "Value",
		},
		{
			name:     "*Value receiver",
			src:      "package test; func (v *Value) Call() {}",
			expected: "Value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.src, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			var funcDecl *ast.FuncDecl
			for _, decl := range file.Decls {
				if fd, ok := decl.(*ast.FuncDecl); ok && fd.Recv != nil {
					funcDecl = fd
					break
				}
			}

			if funcDecl == nil || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
				t.Fatal("Failed to find method receiver")
			}

			receiverType := extractReceiverType(funcDecl.Recv.List[0].Type)
			if receiverType != tt.expected {
				t.Errorf("extractReceiverType() = %q, want %q", receiverType, tt.expected)
			}
		})
	}
}

func TestBuildFormatArg(t *testing.T) {
	tests := []struct {
		name      string
		paramName string
		paramType string
		check     func(*testing.T, ast.Expr)
	}{
		{
			name:      "int",
			paramName: "x",
			paramType: "int",
			check: func(t *testing.T, expr ast.Expr) {
				callExpr, ok := expr.(*ast.CallExpr)
				if !ok {
					t.Fatal("Expected CallExpr")
				}
				selExpr := callExpr.Fun.(*ast.SelectorExpr)
				if selExpr.Sel.Name != "FormatInt" {
					t.Errorf("Expected FormatInt, got %s", selExpr.Sel.Name)
				}
			},
		},
		{
			name:      "string",
			paramName: "s",
			paramType: "string",
			check: func(t *testing.T, expr ast.Expr) {
				callExpr, ok := expr.(*ast.CallExpr)
				if !ok {
					t.Fatal("Expected CallExpr")
				}
				selExpr := callExpr.Fun.(*ast.SelectorExpr)
				if selExpr.Sel.Name != "FormatString" {
					t.Errorf("Expected FormatString, got %s", selExpr.Sel.Name)
				}
			},
		},
		{
			name:      "bytes",
			paramName: "b",
			paramType: "bytes",
			check: func(t *testing.T, expr ast.Expr) {
				callExpr, ok := expr.(*ast.CallExpr)
				if !ok {
					t.Fatal("Expected CallExpr")
				}
				selExpr := callExpr.Fun.(*ast.SelectorExpr)
				if selExpr.Sel.Name != "FormatBytes" {
					t.Errorf("Expected FormatBytes, got %s", selExpr.Sel.Name)
				}
			},
		},
		{
			name:      "slice",
			paramName: "s",
			paramType: "slice",
			check: func(t *testing.T, expr ast.Expr) {
				callExpr, ok := expr.(*ast.CallExpr)
				if !ok {
					t.Fatal("Expected CallExpr")
				}
				selExpr := callExpr.Fun.(*ast.SelectorExpr)
				if selExpr.Sel.Name != "FormatInt" {
					t.Errorf("Expected FormatInt (for len), got %s", selExpr.Sel.Name)
				}
				if lenExpr, ok := callExpr.Args[0].(*ast.CallExpr); ok {
					if lenExpr.Fun.(*ast.Ident).Name != "len" {
						t.Error("Expected len() call for slice")
					}
				}
			},
		},
		{
			name:      "any",
			paramName: "v",
			paramType: "any",
			check: func(t *testing.T, expr ast.Expr) {
				callExpr, ok := expr.(*ast.CallExpr)
				if !ok {
					t.Fatal("Expected CallExpr")
				}
				selExpr := callExpr.Fun.(*ast.SelectorExpr)
				if selExpr.Sel.Name != "FormatAny" {
					t.Errorf("Expected FormatAny, got %s", selExpr.Sel.Name)
				}
				if len(callExpr.Args) != 1 {
					t.Errorf("Expected 1 arg, got %d", len(callExpr.Args))
				}
				if ident, ok := callExpr.Args[0].(*ast.Ident); ok {
					if ident.Name != "v" {
						t.Errorf("Expected arg name 'v', got %s", ident.Name)
					}
				} else {
					t.Error("Expected Ident as arg")
				}
			},
		},
		{
			name:      "interface",
			paramName: "i",
			paramType: "interface",
			check: func(t *testing.T, expr ast.Expr) {
				callExpr, ok := expr.(*ast.CallExpr)
				if !ok {
					t.Fatal("Expected CallExpr")
				}
				selExpr := callExpr.Fun.(*ast.SelectorExpr)
				if selExpr.Sel.Name != "FormatAny" {
					t.Errorf("Expected FormatAny, got %s", selExpr.Sel.Name)
				}
				if len(callExpr.Args) != 1 {
					t.Errorf("Expected 1 arg, got %d", len(callExpr.Args))
				}
				if ident, ok := callExpr.Args[0].(*ast.Ident); ok {
					if ident.Name != "i" {
						t.Errorf("Expected arg name 'i', got %s", ident.Name)
					}
				} else {
					t.Error("Expected Ident as arg")
				}
			},
		},
		{
			name:      "Value type",
			paramName: "v",
			paramType: "Value",
			check: func(t *testing.T, expr ast.Expr) {
				callExpr, ok := expr.(*ast.CallExpr)
				if !ok {
					t.Fatal("Expected CallExpr for Value type")
				}
				selExpr := callExpr.Fun.(*ast.SelectorExpr)
				if selExpr.Sel.Name != "FormatValue" {
					t.Errorf("Expected FormatValue for Value type, got %s", selExpr.Sel.Name)
				}
			},
		},
		{
			name:      "unknown type",
			paramName: "t",
			paramType: "Type",
			check: func(t *testing.T, expr ast.Expr) {
				callExpr, ok := expr.(*ast.CallExpr)
				if !ok {
					t.Fatal("Expected CallExpr for unknown types")
				}
				selExpr := callExpr.Fun.(*ast.SelectorExpr)
				if selExpr.Sel.Name != "FormatAny" {
					t.Errorf("Expected FormatAny for unknown types, got %s", selExpr.Sel.Name)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := buildFormatArg(tt.paramName, tt.paramType)
			if expr == nil {
				t.Fatal("buildFormatArg returned nil")
			}
			tt.check(t, expr)
		})
	}
}

type mockRegistry struct {
	instrumented map[string]map[string]bool
	paths        map[string]string
}

func (m *mockRegistry) IsInstrumented(stdlibPackage, functionName string) bool {
	if pkg, ok := m.instrumented[stdlibPackage]; ok {
		return pkg[functionName]
	}
	return false
}

func (m *mockRegistry) GetInstrumentedImportPath(stdlibPackage string) (string, bool) {
	path, ok := m.paths[stdlibPackage]
	return path, ok
}
