package preprocessor

import (
	"strings"
)

const (
	StdlibSrcPattern       = "/src/"
	StdlibPkgToolPattern   = "go/pkg/tool"
	InstrumentationPattern = "runtime_observe_instrumentation"
	VendorDirPattern       = "/vendor/"
	PkgModDirPattern       = "/pkg/mod/"
)

var DependencyDomainPatterns = []string{
	"github.com/", "gitlab.com/", "bitbucket.org/", "golang.org/x/",
	"google.golang.org/", "gopkg.in/", "go.uber.org/", "k8s.io/",
	"sigs.k8s.io/", "cloud.google.com/", "gocloud.dev/",
}

var StdlibPathPatterns = []string{
	StdlibSrcPattern,
	StdlibPkgToolPattern,
}

var DependencyDirPatterns = []string{
	VendorDirPattern,
	PkgModDirPattern,
}

type InstrumentedPackage struct {
	Pkg       string   `json:"pkg"`
	Functions []string `json:"functions"`
}

type StdlibMethodInstrumentation struct {
	ReceiverType string
	MethodNames  []string
}

type StdlibASTInstrumentation struct {
	PackageName string
	Functions   []string
	Methods     []StdlibMethodInstrumentation
}

type Registry struct {
	Instrumentation    map[string]InstrumentedPackage `json:"instrumentation"`
	SafeStdlibPackages []string                       `json:"safe_stdlib_packages"`
	StdlibAST          map[string]StdlibASTInstrumentation
}

var DefaultRegistry = Registry{
	Instrumentation: map[string]InstrumentedPackage{
		"unsafe": {
			Pkg:       "runtime_observe_instrumentation/unsafe",
			Functions: []string{"Add", "Slice", "SliceData", "String", "StringData"},
		},
	},
	SafeStdlibPackages: []string{
		"encoding/json",
	},
	StdlibAST: map[string]StdlibASTInstrumentation{
		"reflect": {
			PackageName: "reflect",
			Functions: []string{
				"ValueOf",
				"TypeOf",
				"New",
				"NewAt",
				"MakeFunc",
				"MakeMap",
				"MakeMapWithSize",
				"MakeSlice",
				"MakeChan",
			},
			Methods: []StdlibMethodInstrumentation{
				{
					ReceiverType: "Value",
					MethodNames: []string{
						"Call",
						"CallSlice",
						"Method",
						"MethodByName",
						"Set",
						"SetInt",
						"SetString",
						"SetFloat",
						"SetBool",
					},
				},
			},
		},
	},
}

func (r *Registry) IsInstrumented(stdlibPackage, functionName string) bool {
	if pkg, ok := r.Instrumentation[stdlibPackage]; ok {
		for _, fn := range pkg.Functions {
			if fn == functionName {
				return true
			}
		}
	}
	return false
}

func (r *Registry) GetInstrumentedImportPath(stdlibPackage string) (string, bool) {
	if pkg, ok := r.Instrumentation[stdlibPackage]; ok {
		return pkg.Pkg, true
	}
	return "", false
}

func (r *Registry) GetStdlibASTInstrumentation(packageName string) (StdlibASTInstrumentation, bool) {
	instr, ok := r.StdlibAST[packageName]
	return instr, ok
}

func (r *Registry) IsUserPackage(filePath string) bool {
	// User packages are those not in stdlib, not in dependencies
	return !r.IsStdLib(filePath) && !r.IsDependencyPackage(filePath)
}

func (r *Registry) IsStdLib(filePath string) bool {
	for _, pattern := range StdlibPathPatterns {
		if strings.Contains(filePath, pattern) {
			return true
		}
	}

	if strings.Contains(filePath, InstrumentationPattern) && !r.IsDependencyPackage(filePath) {
		return true
	}

	return false
}

func (r *Registry) IsStdLibSafe(filePath string) bool {
	if !r.IsStdLib(filePath) {
		return false
	}

	packageName := extractPackageName(filePath)
	if packageName == "unknown" {
		return false
	}

	for _, pkg := range r.SafeStdlibPackages {
		if pkg == packageName {
			return true
		}
	}

	return false
}

func (r *Registry) IsDependencyPackage(filePath string) bool {
	for _, pattern := range DependencyDomainPatterns {
		if strings.Contains(filePath, pattern) {
			return true
		}
	}

	for _, pattern := range DependencyDirPatterns {
		if strings.Contains(filePath, pattern) {
			return true
		}
	}

	return false
}

func (r *Registry) ShouldInstrument(filePath string) bool {
	if strings.Contains(filePath, InstrumentationPattern) {
		return false
	}

	if r.IsUserPackage(filePath) || r.IsDependencyPackage(filePath) {
		return true
	}

	if r.IsStdLibSafe(filePath) {
		return true
	}

	if r.IsStdLib(filePath) {
		packageName := extractStdlibPackageName(filePath)
		if _, ok := r.StdlibAST[packageName]; ok {
			return true
		}
	}

	return false
}

func extractPackageName(filePath string) string {
	parts := strings.Split(filePath, "/")
	for i, part := range parts {
		if part == "src" && i+1 < len(parts) {
			// Handle case where file is directly under src/
			if i+2 == len(parts) {
				// /path/src/file.go -> return "file" (without .go extension)
				fileName := parts[i+1]
				if strings.HasSuffix(fileName, ".go") {
					return strings.TrimSuffix(fileName, ".go")
				}
				return fileName
			}
			// /path/src/package/file.go -> return "package"
			return strings.Join(parts[i+1:len(parts)-1], "/")
		}
	}

	return "unknown"
}
