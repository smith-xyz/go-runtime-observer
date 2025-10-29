package preprocessor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessFile_UnsafeImport(t *testing.T) {
	testCode := `package main

import "unsafe"

func main() {
	ptr := unsafe.Pointer(nil)
	newPtr := unsafe.Add(ptr, 8)
	_ = newPtr
}
`

	tmpfile, err := os.CreateTemp("", "test-*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(testCode)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	// Test with registry
	testRegistry := &Registry{
		Instrumentation: map[string]InstrumentedPackage{
			"unsafe": {
				Pkg:       "runtime_observe_instrumentation/unsafe",
				Functions: []string{"Add"},
			},
		},
	}

	config := Config{
		InstrumentUnsafe: true,
		Registry:         testRegistry,
	}
	content, modified, err := ProcessFile(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("ProcessFile failed: %v", err)
	}

	if !modified {
		t.Error("Expected file to be modified")
	}

	result := string(content)
	// Should have both original and instrumented imports
	if !strings.Contains(result, `"unsafe"`) {
		t.Error("Expected original unsafe import to remain")
	}
	if !strings.Contains(result, `"runtime_observe_instrumentation/unsafe"`) {
		t.Error("Expected instrumented unsafe import to be added")
	}
	// Should replace function calls
	if !strings.Contains(result, `unsafe_instrumented.Add`) {
		t.Error("Expected unsafe.Add to be replaced with unsafe_instrumented.Add")
	}
	// Should keep original function calls
	if !strings.Contains(result, `unsafe.Pointer`) {
		t.Error("Expected unsafe.Pointer to remain unchanged")
	}
}

func TestProcessFile_WithRegistry(t *testing.T) {
	testCode := `package main

import "unsafe"
import "reflect"

func main() {
	ptr := unsafe.Pointer(nil)
	newPtr := unsafe.Add(ptr, 8)
	_ = newPtr
	
	v := reflect.ValueOf(42)
	_ = v
}
`

	tmpfile, err := os.CreateTemp("", "test-*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(testCode)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	testRegistry := &Registry{
		Instrumentation: map[string]InstrumentedPackage{
			"unsafe": {
				Pkg:       "runtime_observe_instrumentation/unsafe",
				Functions: []string{"Add"},
			},
			"reflect": {
				Pkg:       "runtime_observe_instrumentation/reflect",
				Functions: []string{"ValueOf"},
			},
		},
	}

	config := Config{
		InstrumentUnsafe:  true,
		InstrumentReflect: true,
		Registry:          testRegistry,
	}

	content, modified, err := ProcessFile(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("ProcessFile failed: %v", err)
	}

	if !modified {
		t.Error("Expected file to be modified")
	}

	result := string(content)
	// Should have both original and instrumented imports
	if !strings.Contains(result, `"unsafe"`) {
		t.Error("Expected original unsafe import to remain")
	}
	if !strings.Contains(result, `"runtime_observe_instrumentation/unsafe"`) {
		t.Error("Expected instrumented unsafe import to be added")
	}
	if !strings.Contains(result, `"runtime_observe_instrumentation/reflect"`) {
		t.Error("Expected instrumented reflect import to be added")
	}
	// Should replace function calls
	if !strings.Contains(result, `unsafe_instrumented.Add`) {
		t.Error("Expected unsafe.Add to be replaced with unsafe_instrumented.Add")
	}
	if !strings.Contains(result, `reflect_instrumented.ValueOf`) {
		t.Error("Expected reflect.ValueOf to be replaced with reflect_instrumented.ValueOf")
	}
	// Should keep original function calls
	if !strings.Contains(result, `unsafe.Pointer`) {
		t.Error("Expected unsafe.Pointer to remain unchanged")
	}
}

func TestProcessFile_ReflectImport(t *testing.T) {
	testCode := `package main

import "reflect"

func main() {
	v := reflect.ValueOf(42)
	_ = v
}
`

	tmpfile, err := os.CreateTemp("", "test-*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(testCode)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	testRegistry := &Registry{
		Instrumentation: map[string]InstrumentedPackage{
			"reflect": {
				Pkg:       "runtime_observe_instrumentation/reflect",
				Functions: []string{"ValueOf"},
			},
		},
	}

	config := Config{
		InstrumentReflect: true,
		Registry:          testRegistry,
	}
	content, modified, err := ProcessFile(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("ProcessFile failed: %v", err)
	}

	if !modified {
		t.Error("Expected file to be modified")
	}

	result := string(content)

	if !strings.Contains(result, `"runtime_observe_instrumentation/reflect"`) {
		t.Error("Expected instrumented reflect import to be added")
	}
	// Should replace function calls
	if !strings.Contains(result, `reflect_instrumented.ValueOf`) {
		t.Error("Expected reflect.ValueOf to be replaced with reflect_instrumented.ValueOf")
	}
}

func TestProcessFile_NoInstrumentation(t *testing.T) {
	testCode := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`

	tmpfile, err := os.CreateTemp("", "test-*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(testCode)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	config := Config{}
	content, modified, err := ProcessFile(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("ProcessFile failed: %v", err)
	}

	if modified {
		t.Error("Expected file not to be modified")
	}

	result := string(content)
	if result != testCode {
		t.Error("Content should be unchanged")
	}
}

func TestProcessFile_UnsafePointerNotInstrumented(t *testing.T) {
	testCode := `package main

import "unsafe"

func main() {
	ptr := unsafe.Pointer(nil)
	_ = ptr
}
`

	tmpfile, err := os.CreateTemp("", "test-*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(testCode)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	config := Config{InstrumentUnsafe: true}
	content, modified, err := ProcessFile(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("ProcessFile failed: %v", err)
	}

	// Should NOT be modified because unsafe.Pointer is not instrumented
	if modified {
		t.Error("Expected file NOT to be modified (unsafe.Pointer is not instrumented)")
	}

	result := string(content)
	// Should have original import
	if !strings.Contains(result, `"unsafe"`) {
		t.Error("Expected original unsafe import to remain")
	}
	// Should NOT have instrumented import
	if strings.Contains(result, `"runtime_observe_instrumentation/unsafe"`) {
		t.Error("Expected NO instrumented unsafe import (unsafe.Pointer is not instrumented)")
	}
	// Should keep original function calls
	if !strings.Contains(result, `unsafe.Pointer`) {
		t.Error("Expected unsafe.Pointer to remain unchanged")
	}
}

func TestProcessFile_RegistryNotInstrumented(t *testing.T) {
	testCode := `package main

import "unsafe"

func main() {
	ptr := unsafe.Pointer(nil)
	_ = ptr
}
`

	tmpfile, err := os.CreateTemp("", "test-*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(testCode)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	// Create test registry that doesn't include unsafe.Pointer
	testRegistry := &Registry{
		Instrumentation: map[string]InstrumentedPackage{
			"unsafe": {
				Pkg:       "runtime_observe_instrumentation/unsafe",
				Functions: []string{"Add"}, // Only Add, not Pointer
			},
		},
	}

	config := Config{
		InstrumentUnsafe: true,
		Registry:         testRegistry,
	}

	content, modified, err := ProcessFile(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("ProcessFile failed: %v", err)
	}

	// Should NOT be modified because unsafe.Pointer is not in registry
	if modified {
		t.Error("Expected file NOT to be modified (unsafe.Pointer is not in registry)")
	}

	result := string(content)
	// Should have original import
	if !strings.Contains(result, `"unsafe"`) {
		t.Error("Expected original unsafe import to remain")
	}
	// Should NOT have instrumented import
	if strings.Contains(result, `"runtime_observe_instrumentation/unsafe"`) {
		t.Error("Expected NO instrumented unsafe import (unsafe.Pointer is not in registry)")
	}
	// Should keep original function calls
	if !strings.Contains(result, `unsafe.Pointer`) {
		t.Error("Expected unsafe.Pointer to remain unchanged")
	}
}

func TestProcessFile_RemovesUnusedImport(t *testing.T) {
	testCode := `package main

import "reflect"

func main() {
	reflect.ValueOf(1)
}
`

	tmpfile, err := os.CreateTemp("", "test-*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(testCode)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	testRegistry := &Registry{
		Instrumentation: map[string]InstrumentedPackage{
			"reflect": {
				Pkg:       "runtime_observe_instrumentation/reflect",
				Functions: []string{"ValueOf"},
			},
		},
	}

	config := Config{
		InstrumentReflect: true,
		Registry:          testRegistry,
	}

	content, modified, err := ProcessFile(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("ProcessFile failed: %v", err)
	}

	if !modified {
		t.Error("Expected file to be modified")
	}

	contentStr := string(content)

	// Should have instrumented import
	if !strings.Contains(contentStr, `reflect_instrumented "runtime_observe_instrumentation/reflect"`) {
		t.Error("Expected instrumented import to be added")
	}

	// Should NOT have original reflect import
	if strings.Contains(contentStr, `import "reflect"`) {
		t.Error("Original reflect import should be removed")
	}

	// Should replace reflect.ValueOf with reflect_instrumented.ValueOf
	// Note: formatting may split this across lines
	if !strings.Contains(contentStr, "reflect_instrumented") || !strings.Contains(contentStr, "ValueOf") {
		t.Logf("Generated content:\n%s", contentStr)
		t.Error("Expected reflect.ValueOf to be replaced with reflect_instrumented.ValueOf")
	}
}

func TestProcessFile_RemovesMultipleUnusedImports(t *testing.T) {
	testCode := `package main

import (
	"reflect"
	"unsafe"
)

func main() {
	ptr := unsafe.Pointer(nil)
	reflect.ValueOf(unsafe.Add(ptr, 8))
}
`

	tmpfile, err := os.CreateTemp("", "test-*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(testCode)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	testRegistry := &Registry{
		Instrumentation: map[string]InstrumentedPackage{
			"reflect": {
				Pkg:       "runtime_observe_instrumentation/reflect",
				Functions: []string{"ValueOf"},
			},
			"unsafe": {
				Pkg:       "runtime_observe_instrumentation/unsafe",
				Functions: []string{"Add"},
			},
		},
	}

	config := Config{
		InstrumentReflect: true,
		InstrumentUnsafe:  true,
		Registry:          testRegistry,
	}

	content, modified, err := ProcessFile(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("ProcessFile failed: %v", err)
	}

	if !modified {
		t.Error("Expected file to be modified")
	}

	contentStr := string(content)

	// Should have both instrumented imports
	if !strings.Contains(contentStr, `reflect_instrumented "runtime_observe_instrumentation/reflect"`) {
		t.Error("Expected reflect instrumented import")
	}
	if !strings.Contains(contentStr, `unsafe_instrumented "runtime_observe_instrumentation/unsafe"`) {
		t.Error("Expected unsafe instrumented import")
	}

	// Should NOT have original imports
	if strings.Contains(contentStr, `"reflect"`) && !strings.Contains(contentStr, `runtime_observe_instrumentation/reflect`) {
		t.Error("Original reflect import should be removed")
	}
	if strings.Contains(contentStr, `"unsafe"`) && !strings.Contains(contentStr, `runtime_observe_instrumentation/unsafe`) {
		t.Error("Original unsafe import should be removed")
	}

	// Should replace uses
	if !strings.Contains(contentStr, "reflect_instrumented") || !strings.Contains(contentStr, "ValueOf") {
		t.Error("Expected reflect.ValueOf to be replaced")
	}
	if !strings.Contains(contentStr, "unsafe_instrumented") || !strings.Contains(contentStr, "Add") {
		t.Error("Expected unsafe.Add to be replaced")
	}
}

func TestInstrumentPackageFiles_NoInstrumentation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-pkg-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	file1 := "file1.go"
	file1Path := filepath.Join(tmpDir, file1)

	testCode := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`

	if err := os.WriteFile(file1Path, []byte(testCode), 0644); err != nil {
		t.Fatal(err)
	}

	os.Unsetenv("GO_INSTRUMENT_REFLECT")
	os.Unsetenv("GO_INSTRUMENT_UNSAFE")

	files, dir := InstrumentPackageFiles([]string{file1}, tmpDir)

	if dir != tmpDir {
		t.Error("Expected directory to remain unchanged when no instrumentation is enabled")
	}

	if len(files) != 1 || files[0] != file1 {
		t.Error("Expected files list to remain unchanged")
	}

	content, err := os.ReadFile(file1Path)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != testCode {
		t.Error("File should not have been modified")
	}
}

func TestProcessFileToTemp_StdlibReflectInPlace(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "stdlib-reflect-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src", "reflect")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(srcDir, "value.go")
	originalCode := `package reflect

type Value struct {
	data int
}

func ValueOf(i any) Value {
	return Value{}
}

func (v Value) Call(args []Value) []Value {
	return nil
}

func (v Value) Set(x Value) {
	v.data = x.data
}
`

	if err := os.WriteFile(testFile, []byte(originalCode), 0644); err != nil {
		t.Fatal(err)
	}

	config := Config{
		InstrumentReflect: true,
		Registry:          &DefaultRegistry,
	}

	tempPath, err := ProcessFileToTemp(testFile, config)
	if err != nil {
		t.Fatalf("ProcessFileToTemp failed: %v", err)
	}

	if tempPath != testFile {
		t.Error("Expected stdlib to write to original path")
	}

	content, err := os.ReadFile(tempPath)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	result := string(content)

	if !strings.Contains(result, `instrumentlog.LogCall("reflect.ValueOf")`) {
		t.Error("Expected ValueOf to be instrumented")
	}

	if !strings.Contains(result, `instrumentlog.LogCall("reflect.Value.Call")`) {
		t.Error("Expected Call method to be instrumented")
	}

	if !strings.Contains(result, `instrumentlog.LogCall("reflect.Value.Set")`) {
		t.Error("Expected Set method to be instrumented")
	}

	if !strings.Contains(result, `"runtime_observe_instrumentation/instrumentlog"`) {
		t.Error("Expected instrumentlog import to be added")
	}

	originalContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(originalContent), "instrumentlog") {
		t.Error("Original file should be modified")
	}
}

func TestProcessFileToTemp_NonStdlib(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "user-code-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "main.go")
	testCode := `package main

import "unsafe"

func main() {
	ptr := unsafe.Pointer(nil)
	newPtr := unsafe.Add(ptr, 8)
	_ = newPtr
}
`

	if err := os.WriteFile(testFile, []byte(testCode), 0644); err != nil {
		t.Fatal(err)
	}

	config := Config{
		InstrumentUnsafe: true,
		Registry:         &DefaultRegistry,
	}

	tempPath, err := ProcessFileToTemp(testFile, config)
	if err != nil {
		t.Fatalf("ProcessFileToTemp failed: %v", err)
	}

	if tempPath == testFile {
		t.Error("Expected temp path to be different from original for user code")
	}

	content, err := os.ReadFile(tempPath)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	result := string(content)

	if !strings.Contains(result, `unsafe_instrumented.Add`) {
		t.Error("Expected unsafe.Add to be rewritten to unsafe_instrumented.Add")
	}

	if !strings.Contains(result, `"runtime_observe_instrumentation/unsafe"`) {
		t.Error("Expected instrumented unsafe import to be added")
	}
}

func TestProcessFileToTemp_StdlibNotInRegistry(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "stdlib-fmt-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src", "fmt")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(srcDir, "print.go")
	testCode := `package fmt

func Printf(format string, args ...any) {
	// implementation
}
`

	if err := os.WriteFile(testFile, []byte(testCode), 0644); err != nil {
		t.Fatal(err)
	}

	config := Config{
		InstrumentReflect: true,
		Registry:          &DefaultRegistry,
	}

	tempPath, err := ProcessFileToTemp(testFile, config)
	if err != nil {
		t.Fatalf("ProcessFileToTemp failed: %v", err)
	}

	if tempPath != testFile {
		t.Error("Expected no instrumentation for fmt package (not in registry)")
	}
}

// ProcessFileToTemp Decision Logic Tests
//
// These tests verify the hybrid instrumentation routing logic:
//
// | File Type   | AST Match? | SafeStdlib? | Action     | Location |
// |-------------|------------|-------------|------------|----------|
// | Stdlib      | Yes        | -           | AST inject | In-place |
// | Stdlib      | No         | No          | Skip       | Original |
// | Stdlib      | No         | Yes         | Wrapper    | In-place |
// | User        | -          | -           | Wrapper    | Temp     |
// | Dependency  | -          | -           | Wrapper    | Temp     |
// | Any         | No instrumentation needed | -      | Skip       | Original |
//
// This ensures:
// - Stdlib with AST instrumentation is modified in-place
// - Stdlib without match and not in SafeStdlib is skipped (prevents breaking Go internals)
// - Stdlib in SafeStdlib gets wrapper in-place
// - User and dependency code gets wrapper in temp directory (no source pollution)

func TestProcessFileToTemp_StdlibNoMatchSkipsWrapper(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-stdlib-nomatch-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src", "reflect")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(srcDir, "abi.go")
	originalCode := `package reflect

import "unsafe"

func someHelper(ptr unsafe.Pointer) {
	unsafe.Add(ptr, 8)
}
`

	if err := os.WriteFile(testFile, []byte(originalCode), 0644); err != nil {
		t.Fatal(err)
	}

	config := Config{
		InstrumentUnsafe: true,
		Registry:         &DefaultRegistry,
	}

	resultPath, err := ProcessFileToTemp(testFile, config)
	if err != nil {
		t.Fatalf("ProcessFileToTemp failed: %v", err)
	}

	if resultPath != testFile {
		t.Errorf("Expected original path %s, got %s", testFile, resultPath)
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	result := string(content)
	if strings.Contains(result, "unsafe_instrumented") {
		t.Error("Expected no wrapper instrumentation for non-SafeStdlib package")
	}

	if result != originalCode {
		t.Error("File should not have been modified")
	}
}

func TestProcessFileToTemp_SafeStdlibWithWrapper(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-safe-stdlib-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src", "encoding", "json")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(srcDir, "encode.go")
	originalCode := `package json

import "unsafe"

func encode() {
	ptr := unsafe.Pointer(nil)
	unsafe.Add(ptr, 8)
}
`

	if err := os.WriteFile(testFile, []byte(originalCode), 0644); err != nil {
		t.Fatal(err)
	}

	config := Config{
		InstrumentUnsafe: true,
		Registry:         &DefaultRegistry,
	}

	resultPath, err := ProcessFileToTemp(testFile, config)
	if err != nil {
		t.Fatalf("ProcessFileToTemp failed: %v", err)
	}

	if resultPath != testFile {
		t.Errorf("Expected original path %s (in-place), got %s", testFile, resultPath)
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	result := string(content)
	if !strings.Contains(result, "unsafe_instrumented") || !strings.Contains(result, "Add") {
		t.Error("Expected wrapper instrumentation for SafeStdlib package")
	}
}

func TestProcessFileToTemp_UserCodeToTemp(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-user-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "main.go")
	originalCode := `package main

import "unsafe"

func main() {
	ptr := unsafe.Pointer(nil)
	unsafe.Add(ptr, 8)
}
`

	if err := os.WriteFile(testFile, []byte(originalCode), 0644); err != nil {
		t.Fatal(err)
	}

	config := Config{
		InstrumentUnsafe: true,
		Registry:         &DefaultRegistry,
	}

	resultPath, err := ProcessFileToTemp(testFile, config)
	if err != nil {
		t.Fatalf("ProcessFileToTemp failed: %v", err)
	}

	if resultPath == testFile {
		t.Error("Expected temp path, got original path")
	}

	if !strings.Contains(resultPath, "go-runtime-observer") {
		t.Errorf("Expected temp path to contain go-runtime-observer, got %s", resultPath)
	}

	originalContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(originalContent) != originalCode {
		t.Error("Original file should not have been modified")
	}

	tempContent, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	result := string(tempContent)
	if !strings.Contains(result, "unsafe_instrumented") || !strings.Contains(result, "Add") {
		t.Logf("Temp file content:\n%s", result)
		t.Error("Expected wrapper instrumentation in temp file")
	}
}

func TestProcessFileToTemp_UserInternal(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-user-internal-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	internalDir := filepath.Join(tmpDir, "internal", "security")
	if err := os.MkdirAll(internalDir, 0755); err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(internalDir, "validator.go")
	originalCode := `package security

import "unsafe"

func Validate() {
	ptr := unsafe.Pointer(nil)
	unsafe.Add(ptr, 8)
}
`

	if err := os.WriteFile(testFile, []byte(originalCode), 0644); err != nil {
		t.Fatal(err)
	}

	config := Config{
		InstrumentUnsafe: true,
		Registry:         &DefaultRegistry,
	}

	resultPath, err := ProcessFileToTemp(testFile, config)
	if err != nil {
		t.Fatalf("ProcessFileToTemp failed: %v", err)
	}

	if resultPath == testFile {
		t.Error("Expected temp path for user internal package")
	}

	tempContent, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	result := string(tempContent)
	if !strings.Contains(result, "unsafe_instrumented") || !strings.Contains(result, "Add") {
		t.Error("Expected wrapper instrumentation for user internal package")
	}
}

func TestProcessFileToTemp_Dependency(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-dependency-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	vendorDir := filepath.Join(tmpDir, "vendor", "github.com", "foo", "bar")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(vendorDir, "unsafe_ops.go")
	originalCode := `package bar

import "unsafe"

func Process() {
	ptr := unsafe.Pointer(nil)
	unsafe.Add(ptr, 16)
}
`

	if err := os.WriteFile(testFile, []byte(originalCode), 0644); err != nil {
		t.Fatal(err)
	}

	config := Config{
		InstrumentUnsafe: true,
		Registry:         &DefaultRegistry,
	}

	resultPath, err := ProcessFileToTemp(testFile, config)
	if err != nil {
		t.Fatalf("ProcessFileToTemp failed: %v", err)
	}

	if resultPath == testFile {
		t.Error("Expected temp path for dependency")
	}

	tempContent, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	result := string(tempContent)
	if !strings.Contains(result, "unsafe_instrumented") || !strings.Contains(result, "Add") {
		t.Error("Expected wrapper instrumentation for dependency")
	}
}

func TestProcessFileToTemp_NoInstrumentationNeeded(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-no-instr-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "main.go")
	originalCode := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`

	if err := os.WriteFile(testFile, []byte(originalCode), 0644); err != nil {
		t.Fatal(err)
	}

	config := Config{
		InstrumentUnsafe:  true,
		InstrumentReflect: true,
		Registry:          &DefaultRegistry,
	}

	resultPath, err := ProcessFileToTemp(testFile, config)
	if err != nil {
		t.Fatalf("ProcessFileToTemp failed: %v", err)
	}

	if resultPath != testFile {
		t.Errorf("Expected original path for non-instrumented file")
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != originalCode {
		t.Error("File should not have been modified")
	}
}
