package preprocessor

import (
	"strings"

	"github.com/smith-xyz/go-runtime-observer/pkg/preprocessor/types"
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

type Registry struct {
	Instrumentation    map[string]types.InstrumentedPackage `json:"instrumentation"`
	SafeStdlibPackages []string                             `json:"safe_stdlib_packages"`
	StdlibAST          map[string]types.StdlibASTInstrumentation
	ExcludedPackages   []string `json:"excluded_packages"`
}

var DefaultRegistry = Registry{
	Instrumentation: map[string]types.InstrumentedPackage{
		"unsafe": {
			Pkg:       "runtime_observe_instrumentation/unsafe",
			Functions: []string{"Add", "Slice", "SliceData", "String", "StringData"},
		},
	},
	SafeStdlibPackages: []string{
		"encoding/json",
	},
	ExcludedPackages: []string{},
	StdlibAST: map[string]types.StdlibASTInstrumentation{
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
			Methods: []types.StdlibMethodInstrumentation{
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
					// MethodByName and Method return reflect.Value instances that are later used in Call()
					// We record correlations so we can connect MethodByName("GetName") -> Call() in logs
					CorrelationRecordingMethods: []string{"MethodByName", "Method"},
					// MethodByName takes a "name" parameter we want to record; Method takes an index from its return call
					MethodIdentifierExtractors: map[string]string{
						"MethodByName": "param:name",
						"Method":       "call:0",
					},
					// MethodByName internally calls v.Method(index), so we check for "Method" in its return statements
					// Method returns itself or a direct Value{}, so empty slice means check for same method name
					ReturnExpressionMethods: map[string][]string{
						"MethodByName": {"Method"},
						"Method":       {},
					},
					// Call and CallSlice consume correlations recorded by MethodByName/Method
					CorrelationLookupMethods: []string{"Call", "CallSlice"},
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

func (r *Registry) GetStdlibASTInstrumentation(packageName string) (types.StdlibASTInstrumentation, bool) {
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

	if r.IsExcludedPackage(filePath) {
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

func (r *Registry) IsExcludedPackage(filePath string) bool {
	for _, excluded := range r.ExcludedPackages {
		if strings.Contains(filePath, excluded) {
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
