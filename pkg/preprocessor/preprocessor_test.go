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
		SafeStdlibPackages: []string{"main"},
	}
	
	config := Config{
		InstrumentUnsafe: true,
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
		SafeStdlibPackages: []string{"main"},
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
		SafeStdlibPackages: []string{"main"},
	}
	
	config := Config{
		InstrumentReflect: true,
		Registry:           testRegistry,
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
		SafeStdlibPackages: []string{"main"},
	}
	
	config := Config{
		InstrumentUnsafe: true,
		Registry:          testRegistry,
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
		SafeStdlibPackages: []string{"main"},
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
		SafeStdlibPackages: []string{"main"},
	}
	
	config := Config{
		InstrumentReflect: true,
		InstrumentUnsafe: true,
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

func TestInstrumentPackageFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-pkg-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	file1 := "file1.go"
	file1Path := filepath.Join(tmpDir, file1)

	testCode := `package main

import "reflect"

func main() {
	reflect.ValueOf(42)
}
`

	if err := os.WriteFile(file1Path, []byte(testCode), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GO_INSTRUMENT_REFLECT", "true")
	defer os.Unsetenv("GO_INSTRUMENT_REFLECT")

	files, dir := InstrumentPackageFiles([]string{file1}, tmpDir)

	if dir == tmpDir {
		t.Error("Expected instrumented directory to be different from original")
	}

	instrumentedPath := filepath.Join(dir, files[0])
	content, err := os.ReadFile(instrumentedPath)
	if err != nil {
		t.Fatalf("Failed to read instrumented file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "runtime_observe_instrumentation/reflect") {
		t.Error("Expected instrumented import to be added")
	}

	originalContent, err := os.ReadFile(file1Path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(originalContent), "runtime_observe_instrumentation/reflect") {
		t.Error("Original file should not be modified")
	}

	_ = CleanupTempDir()
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

func TestInstrumentPackageFiles_MultipleFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-pkg-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	file1Path := filepath.Join(tmpDir, "file1.go")
	file2Path := filepath.Join(tmpDir, "file2.go")

	testCode1 := `package main
import "reflect"
func test1() { reflect.ValueOf(1) }
`

	testCode2 := `package main
import "unsafe"
func test2() { unsafe.Add(nil, 8) }
`

	if err := os.WriteFile(file1Path, []byte(testCode1), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(file2Path, []byte(testCode2), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GO_INSTRUMENT_REFLECT", "true")
	os.Setenv("GO_INSTRUMENT_UNSAFE", "true")
	defer func() {
		os.Unsetenv("GO_INSTRUMENT_REFLECT")
		os.Unsetenv("GO_INSTRUMENT_UNSAFE")
		_ = CleanupTempDir()
	}()

	files, dir := InstrumentPackageFiles([]string{"file1.go", "file2.go"}, tmpDir)

	if dir == tmpDir {
		t.Error("Expected instrumented directory to be different from original")
	}

	file1InstrumentedPath := filepath.Join(dir, "file1.go")
	file2InstrumentedPath := filepath.Join(dir, "file2.go")

	content1, err := os.ReadFile(file1InstrumentedPath)
	if err != nil {
		t.Fatalf("Failed to read instrumented file1: %v", err)
	}

	content2, err := os.ReadFile(file2InstrumentedPath)
	if err != nil {
		t.Fatalf("Failed to read instrumented file2: %v", err)
	}

	if !strings.Contains(string(content1), "runtime_observe_instrumentation/reflect") {
		t.Error("Expected file1 to have reflect instrumentation")
	}

	if !strings.Contains(string(content2), "runtime_observe_instrumentation/unsafe") {
		t.Error("Expected file2 to have unsafe instrumentation")
	}

	origContent1, _ := os.ReadFile(file1Path)
	origContent2, _ := os.ReadFile(file2Path)

	if strings.Contains(string(origContent1), "runtime_observe_instrumentation") {
		t.Error("Original file1 should not be modified")
	}

	if strings.Contains(string(origContent2), "runtime_observe_instrumentation") {
		t.Error("Original file2 should not be modified")
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}
}
