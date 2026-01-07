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
			Logger:      types.LoggerTypeInstrument,
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
					CorrelationRecordingMethods: []string{"MethodByName", "Method"},
					MethodIdentifierExtractors: map[string]string{
						"MethodByName": "param:name",
						"Method":       "call:0",
					},
					ReturnExpressionMethods: map[string][]string{
						"MethodByName": {"Method"},
						"Method":       {},
					},
					CorrelationLookupMethods: []string{"Call", "CallSlice"},
				},
			},
		},
		"crypto/md5": {
			PackageName: "crypto/md5",
			Logger:      types.LoggerTypeFormat,
			Functions:   []string{"New", "Sum"},
		},
		"crypto/sha1": {
			PackageName: "crypto/sha1",
			Logger:      types.LoggerTypeFormat,
			Functions:   []string{"New", "Sum"},
		},
		"crypto/sha256": {
			PackageName: "crypto/sha256",
			Logger:      types.LoggerTypeFormat,
			Functions:   []string{"New", "New224", "Sum256", "Sum224"},
		},
		"crypto/sha512": {
			PackageName: "crypto/sha512",
			Logger:      types.LoggerTypeFormat,
			Functions:   []string{"New", "New384", "New512_224", "New512_256", "Sum512", "Sum384", "Sum512_224", "Sum512_256"},
		},
		"crypto/aes": {
			PackageName: "crypto/aes",
			Logger:      types.LoggerTypeFormat,
			Functions:   []string{"NewCipher"},
		},
		"crypto/des": {
			PackageName: "crypto/des",
			Logger:      types.LoggerTypeFormat,
			Functions:   []string{"NewCipher", "NewTripleDESCipher"},
		},
		"crypto/rsa": {
			PackageName: "crypto/rsa",
			Logger:      types.LoggerTypeFormat,
			Functions:   []string{"GenerateKey", "EncryptPKCS1v15", "DecryptPKCS1v15", "SignPKCS1v15", "VerifyPKCS1v15", "EncryptOAEP", "DecryptOAEP", "SignPSS", "VerifyPSS"},
		},
		"crypto/ecdsa": {
			PackageName: "crypto/ecdsa",
			Logger:      types.LoggerTypeFormat,
			Functions:   []string{"GenerateKey", "Sign", "SignASN1", "Verify", "VerifyASN1"},
		},
		"crypto/ed25519": {
			PackageName: "crypto/ed25519",
			Logger:      types.LoggerTypeFormat,
			Functions:   []string{"GenerateKey", "Sign", "Verify"},
		},
		"crypto/tls": {
			PackageName: "crypto/tls",
			Logger:      types.LoggerTypeFormat,
			Functions:   []string{"Dial", "DialWithDialer", "Client", "Server", "Listen", "NewListener"},
		},
		"crypto/x509": {
			PackageName: "crypto/x509",
			Logger:      types.LoggerTypeFormat,
			Functions:   []string{"ParseCertificate", "ParseCertificates", "ParseCertificateRequest", "CreateCertificate", "CreateCertificateRequest", "ParsePKCS1PrivateKey", "ParsePKCS8PrivateKey", "ParseECPrivateKey"},
		},
		"crypto/rand": {
			PackageName: "crypto/rand",
			Logger:      types.LoggerTypeFormat,
			Functions:   []string{"Read", "Prime", "Int"},
		},
		"crypto/hmac": {
			PackageName: "crypto/hmac",
			Logger:      types.LoggerTypeFormat,
			Functions:   []string{"New", "Equal"},
		},
		"math/rand": {
			PackageName: "math/rand",
			Logger:      types.LoggerTypeFormat,
			Functions:   []string{"Read", "Int", "Intn", "Int31", "Int31n", "Int63", "Int63n", "Uint32", "Uint64", "Float32", "Float64"},
		},
		"crypto/rc4": {
			PackageName: "crypto/rc4",
			Logger:      types.LoggerTypeFormat,
			Functions:   []string{"NewCipher"},
		},
		"crypto/dsa": {
			PackageName: "crypto/dsa",
			Logger:      types.LoggerTypeFormat,
			Functions:   []string{"GenerateParameters", "GenerateKey", "Sign", "Verify"},
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
