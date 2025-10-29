package preprocessor

import (
	"testing"
)

func TestRegistry_IsUserPackage(t *testing.T) {
	registry := &DefaultRegistry

	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		// User code - should return true
		{"user main.go", "/home/user/myproject/main.go", true},
		{"user package", "/home/user/myproject/pkg/mypackage/file.go", true},
		{"user internal", "/home/user/myproject/internal/helper/file.go", true},
		{"user cmd", "/home/user/myproject/cmd/server/main.go", true},
		{"user api", "/home/user/myproject/api/handlers.go", true},
		{"user service", "/home/user/myproject/services/auth.go", true},
		{"user model", "/home/user/myproject/models/user.go", true},

		// Go stdlib - should return false
		{"go stdlib encoding/json", "/tmp/go-source/1.23.0/go/src/encoding/json/encode.go", false},
		{"go stdlib runtime", "/tmp/go-source/1.23.0/go/src/runtime/malloc.go", false},
		{"go stdlib internal", "/tmp/go-source/1.23.0/go/src/internal/reflectlite/type.go", false},
		{"go stdlib os", "/tmp/go-source/1.23.0/go/src/os/file.go", false},
		{"go stdlib net/http", "/tmp/go-source/1.23.0/go/src/net/http/server.go", false},

		// Dependencies - should return false (handled by IsDependencyPackage)
		{"github dependency", "/home/user/myproject/vendor/github.com/gin-gonic/gin/gin.go", false},
		{"golang.org dependency", "/home/user/myproject/pkg/mod/golang.org/x/net/http2/server.go", false},
		{"go mod cache", "/home/user/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.IsUserPackage(tt.filePath)
			if result != tt.expected {
				t.Errorf("IsUserPackage(%q) = %v, expected %v", tt.filePath, result, tt.expected)
			}
		})
	}
}

func TestRegistry_IsDependencyPackage(t *testing.T) {
	registry := &DefaultRegistry

	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		// Dependencies - should return true
		{"github dependency", "/home/user/myproject/vendor/github.com/gin-gonic/gin/gin.go", true},
		{"gitlab dependency", "/home/user/myproject/vendor/gitlab.com/user/repo/file.go", true},
		{"golang.org dependency", "/home/user/myproject/pkg/mod/golang.org/x/net/http2/server.go", true},
		{"google.golang.org dependency", "/home/user/myproject/vendor/google.golang.org/grpc/server.go", true},
		{"gopkg.in dependency", "/home/user/myproject/vendor/gopkg.in/yaml.v2/yaml.go", true},
		{"k8s.io dependency", "/home/user/myproject/vendor/k8s.io/client-go/kubernetes/clientset.go", true},
		{"vendor directory", "/home/user/myproject/vendor/some/package/file.go", true},
		{"pkg/mod directory", "/home/user/myproject/pkg/mod/some/package/file.go", true},
		{"go mod cache", "/home/user/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go", true},
		{"go mod cache golang.org", "/home/user/go/pkg/mod/golang.org/x/net@v0.17.0/http2/server.go", true},
		{"go mod cache google", "/home/user/go/pkg/mod/google.golang.org/grpc@v1.59.0/server.go", true},

		// User code - should return false
		{"user main.go", "/home/user/myproject/main.go", false},
		{"user package", "/home/user/myproject/pkg/mypackage/file.go", false},
		{"user internal", "/home/user/myproject/internal/helper/file.go", false},
		{"user cmd", "/home/user/myproject/cmd/server/main.go", false},
		{"user api", "/home/user/myproject/api/handlers.go", false},

		// Go stdlib - should return false
		{"go stdlib", "/tmp/go-source/1.23.0/go/src/encoding/json/encode.go", false},
		{"go runtime", "/tmp/go-source/1.23.0/go/src/runtime/malloc.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.IsDependencyPackage(tt.filePath)
			if result != tt.expected {
				t.Errorf("IsDependencyPackage(%q) = %v, expected %v", tt.filePath, result, tt.expected)
			}
		})
	}
}

func TestRegistry_IsStdLibSafe(t *testing.T) {
	registry := &DefaultRegistry

	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		// Safe stdlib packages - should return true
		{"encoding/json", "/tmp/go-source/1.23.0/go/src/encoding/json/encode.go", true},

		// Unsafe stdlib packages - should return false
		{"runtime package", "/tmp/go-source/1.23.0/go/src/runtime/malloc.go", false},
		{"internal package", "/tmp/go-source/1.23.0/go/src/internal/reflectlite/type.go", false},
		{"unsafe package", "/tmp/go-source/1.23.0/go/src/unsafe/unsafe.go", false},
		{"syscall package", "/tmp/go-source/1.23.0/go/src/syscall/syscall_unix.go", false},
		{"unknown stdlib", "/tmp/go-source/1.23.0/go/src/unknown/package/file.go", false},
		{"os package", "/tmp/go-source/1.23.0/go/src/os/file.go", false},
		{"net/http package", "/tmp/go-source/1.23.0/go/src/net/http/server.go", false},

		// Non-stdlib - should return false
		{"user code", "/home/user/myproject/main.go", false},
		{"dependency", "/home/user/myproject/vendor/github.com/gin-gonic/gin/gin.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.IsStdLibSafe(tt.filePath)
			if result != tt.expected {
				t.Errorf("IsStdLibSafe(%q) = %v, expected %v", tt.filePath, result, tt.expected)
			}
		})
	}
}

func TestRegistry_ShouldInstrument(t *testing.T) {
	registry := &DefaultRegistry

	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		// User code - should return true
		{"user main.go", "/home/user/myproject/main.go", true},
		{"user package", "/home/user/myproject/pkg/mypackage/file.go", true},
		{"user internal", "/home/user/myproject/internal/helper/file.go", true},
		{"user cmd", "/home/user/myproject/cmd/server/main.go", true},
		{"user api", "/home/user/myproject/api/handlers.go", true},

		// Dependencies - should return true
		{"github dependency", "/home/user/myproject/vendor/github.com/gin-gonic/gin/gin.go", true},
		{"golang.org dependency", "/home/user/myproject/pkg/mod/golang.org/x/net/http2/server.go", true},
		{"go mod cache", "/home/user/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go", true},

		// Safe stdlib - should return true
		{"encoding/json", "/tmp/go-source/1.23.0/go/src/encoding/json/encode.go", true},

		// Unsafe stdlib - should return false
		{"runtime package", "/tmp/go-source/1.23.0/go/src/runtime/malloc.go", false},
		{"internal package", "/tmp/go-source/1.23.0/go/src/internal/reflectlite/type.go", false},
		{"unsafe package", "/tmp/go-source/1.23.0/go/src/unsafe/unsafe.go", false},
		{"syscall package", "/tmp/go-source/1.23.0/go/src/syscall/syscall_unix.go", false},
		{"unknown stdlib", "/tmp/go-source/1.23.0/go/src/unknown/package/file.go", false},
		{"os package", "/tmp/go-source/1.23.0/go/src/os/file.go", false},
		{"net/http package", "/tmp/go-source/1.23.0/go/src/net/http/server.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.ShouldInstrument(tt.filePath)
			if result != tt.expected {
				t.Errorf("ShouldInstrument(%q) = %v, expected %v", tt.filePath, result, tt.expected)
			}
		})
	}
}

func TestRegistry_ExtractPackageName(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected string
	}{
		// Stdlib packages
		{"encoding/json", "/tmp/go-source/1.23.0/go/src/encoding/json/encode.go", "encoding/json"},
		{"encoding/xml", "/tmp/go-source/1.23.0/go/src/encoding/xml/marshal.go", "encoding/xml"},
		{"crypto/sha256", "/tmp/go-source/1.23.0/go/src/crypto/sha256/sha256.go", "crypto/sha256"},
		{"os", "/tmp/go-source/1.23.0/go/src/os/file.go", "os"},
		{"net/http", "/tmp/go-source/1.23.0/go/src/net/http/server.go", "net/http"},
		{"runtime", "/tmp/go-source/1.23.0/go/src/runtime/malloc.go", "runtime"},
		{"internal/reflectlite", "/tmp/go-source/1.23.0/go/src/internal/reflectlite/type.go", "internal/reflectlite"},
		{"unsafe", "/tmp/go-source/1.23.0/go/src/unsafe/unsafe.go", "unsafe"},
		{"syscall", "/tmp/go-source/1.23.0/go/src/syscall/syscall_unix.go", "syscall"},
		{"file directly under src", "/tmp/go-source/1.23.0/go/src/unsafe.go", "unsafe"},
		{"file directly under src no extension", "/tmp/go-source/1.23.0/go/src/unsafe", "unsafe"},

		// Non-stdlib - should return unknown
		{"user code", "/home/user/myproject/main.go", "unknown"},
		{"dependency", "/home/user/myproject/vendor/github.com/gin-gonic/gin/gin.go", "unknown"},
		{"go mod cache", "/home/user/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go", "unknown"},
		{"no src", "/home/user/some/file.go", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPackageName(tt.filePath)
			if result != tt.expected {
				t.Errorf("extractPackageName(%q) = %q, expected %q", tt.filePath, result, tt.expected)
			}
		})
	}
}
