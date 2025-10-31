package preprocessor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessStdlibFile_Reflect(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "stdlib-test-*")
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

	content, modified, err := ProcessStdlibFile(testFile, &DefaultRegistry)
	if err != nil {
		t.Fatalf("ProcessStdlibFile failed: %v", err)
	}

	if !modified {
		t.Error("Expected file to be modified")
	}

	result := string(content)

	if !strings.Contains(result, `instrumentlog.LogCall("reflect.ValueOf"`) {
		t.Error("Expected ValueOf function to be instrumented")
	}

	if !strings.Contains(result, `instrumentlog.LogCall("reflect.Value.Call"`) {
		t.Error("Expected Call method to be instrumented")
	}

	if !strings.Contains(result, `instrumentlog.LogCall("reflect.Value.Set"`) {
		t.Error("Expected Set method to be instrumented")
	}

	if !strings.Contains(result, `"runtime_observe_instrumentation/instrumentlog"`) {
		t.Error("Expected instrumentlog import to be added")
	}
}

func TestProcessStdlibFile_NonReflect(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "stdlib-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src", "fmt")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(srcDir, "print.go")
	originalCode := `package fmt

func Printf(format string, args ...any) {
	// implementation
}
`

	if err := os.WriteFile(testFile, []byte(originalCode), 0644); err != nil {
		t.Fatal(err)
	}

	content, modified, err := ProcessStdlibFile(testFile, &DefaultRegistry)
	if err != nil {
		t.Fatalf("ProcessStdlibFile failed: %v", err)
	}

	if modified {
		t.Error("Did not expect fmt package to be modified")
	}

	if content != nil {
		t.Error("Expected nil content for unmodified file")
	}
}

func TestExtractStdlibPackageName(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected string
	}{
		{
			name:     "reflect package",
			filePath: "/path/to/go/src/reflect/value.go",
			expected: "reflect",
		},
		{
			name:     "net/http package",
			filePath: "/path/to/go/src/net/http/server.go",
			expected: "net",
		},
		{
			name:     "instrumentation package",
			filePath: "/path/to/go/src/runtime_observe_instrumentation/instrumentlog/logger.go",
			expected: "",
		},
		{
			name:     "non-stdlib package",
			filePath: "/path/to/project/pkg/mypackage/file.go",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractStdlibPackageName(tt.filePath)
			if result != tt.expected {
				t.Errorf("extractStdlibPackageName(%q) = %q, want %q", tt.filePath, result, tt.expected)
			}
		})
	}
}
