package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions"
	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions/config"
)

const testGoVersion = "test-1.0.0"

func init() {
	versions.SupportedVersions[testGoVersion] = config.VersionConfig{
		Go:    testGoVersion,
		Notes: "Test version",
		Injections: []config.InjectionConfig{
			{
				Name:        "dependency",
				TargetFile:  "test/pkg.go",
				Line:        32,
				Description: "Test dependency injection",
				Instrument: config.InstrumentCall{
					Function: "InstrumentPackageFiles",
					Args:     []string{"data.p.GoFiles", "data.p.Dir"},
					Result:   []string{"data.p.GoFiles", "data.p.Dir"},
				},
				Reparse: config.ReparseCall{
					Result:   []string{"data.p", "data.err"},
					Function: "buildContext.ImportDir",
					Args:     []string{"data.p.Dir", "buildMode"},
				},
			},
			{
				Name:        "command_line",
				TargetFile:  "test/pkg.go",
				Line:        40,
				Description: "Test command-line injection",
				Instrument: config.InstrumentCall{
					Function: "InstrumentPackageFiles",
					Args:     []string{"bp.GoFiles", "dir"},
					Result:   []string{"bp.GoFiles", "dir"},
				},
				Reparse: config.ReparseCall{
					Result:   []string{"bp", "err"},
					Function: "ctxt.ImportDir",
					Args:     []string{"dir", "0"},
				},
			},
		},
	}
}

func TestInstrumentFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-go-source-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	pkgDir := filepath.Join(tmpDir, "test")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	pkgFile := filepath.Join(pkgDir, "pkg.go")
	testContent := `package load

import (
	"go/build"
)

type Package struct {
	GoFiles []string
	Dir string
}
	
type packageData struct {
	p *Package
	err error
}

type buildContextType struct{}
func (buildContextType) ImportDir(dir string, mode int) (*Package, error) { return nil, nil }

type ctxtType struct{}
func (ctxtType) ImportDir(dir string, mode int) (*Package, error) { return nil, nil }

var buildContext buildContextType
var ctxt ctxtType

func loadPackage() {
	data := packageData{p: nil, err: nil}
	r := struct{ dir string }{dir: "."}
	buildMode := 0
	goto Happy
	
Happy:
	_ = data
	_ = r
	_ = buildMode
}

func goFilesPackage() {
	dir := "."
	bp, err := ctxt.ImportDir(dir, 0)
	pkg := new(Package)
	_ = bp
	_ = err
	_ = pkg
}
`
	if err := os.WriteFile(pkgFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	intermediateContent, _ := os.ReadFile(pkgFile)
	lines := strings.Split(string(intermediateContent), "\n")
	for i, line := range lines {
		if strings.Contains(line, "Happy:") {
			t.Logf("Found Happy: at line %d", i+1)
		}
		if strings.Contains(line, "bp, err") && strings.Contains(line, "ctxt.ImportDir") {
			t.Logf("Found bp, err = ctxt.ImportDir at line %d", i+1)
		}
	}

	if err = InstrumentFile(tmpDir, testGoVersion); err != nil {
		modifiedContent, _ := os.ReadFile(pkgFile)
		t.Logf("Generated file content:\n%s\n", string(modifiedContent))
		t.Fatalf("InstrumentFile failed: %v", err)
	}

	modifiedContent, err := os.ReadFile(pkgFile)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}

	content := string(modifiedContent)

	if !strings.Contains(content, IMPORT_VALUE) {
		t.Error("Import was not added")
	}

	if !strings.Contains(content, "preprocessor.InstrumentPackageFiles") {
		t.Error("InstrumentPackageFiles call was not added")
	}

	if !strings.Contains(content, "data.p, data.err = buildContext.ImportDir") {
		t.Error("Dependency re-parse was not added")
	}

	if !strings.Contains(content, "bp, err = ctxt.ImportDir") {
		t.Error("Command-line re-parse was not added")
	}
}

func TestApplyPatch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "patch-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFilePath := filepath.Join(tmpDir, "test.go")
	originalContent := `package test

var (
	BuildBuildvcs      = "auto"
	BuildMode          = "default"
)
`
	if err := os.WriteFile(testFilePath, []byte(originalContent), 0644); err != nil {
		t.Fatal(err)
	}

	patch := config.PatchConfig{
		Name:        "buildvcs_test",
		TargetFile:  "test.go",
		Description: "Test patch for buildvcs",
		Find:        `BuildBuildvcs      = "auto"`,
		Replace:     `BuildBuildvcs      = "false"`,
	}

	if err := ApplyPatch(testFilePath, patch); err != nil {
		t.Fatalf("ApplyPatch failed: %v", err)
	}

	modifiedContent, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(modifiedContent), `BuildBuildvcs      = "false"`) {
		t.Error("Patch was not applied correctly")
	}

	if strings.Contains(string(modifiedContent), `BuildBuildvcs      = "auto"`) {
		t.Error("Original string still exists after patching")
	}
}

func TestApplyPatch_Idempotent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "patch-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFilePath := filepath.Join(tmpDir, "test.go")
	alreadyPatchedContent := `package test

var (
	BuildBuildvcs      = "false"
	BuildMode          = "default"
)
`
	if err := os.WriteFile(testFilePath, []byte(alreadyPatchedContent), 0644); err != nil {
		t.Fatal(err)
	}

	patch := config.PatchConfig{
		Name:        "buildvcs_test",
		TargetFile:  "test.go",
		Description: "Test patch for buildvcs",
		Find:        `BuildBuildvcs      = "auto"`,
		Replace:     `BuildBuildvcs      = "false"`,
	}

	if err := ApplyPatch(testFilePath, patch); err != nil {
		t.Fatalf("ApplyPatch should be idempotent but failed: %v", err)
	}

	modifiedContent, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatal(err)
	}

	if string(modifiedContent) != alreadyPatchedContent {
		t.Error("Already patched file was modified")
	}
}

func TestApplyPatch_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "patch-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFilePath := filepath.Join(tmpDir, "test.go")
	originalContent := `package test

var (
	BuildMode = "default"
)
`
	if err := os.WriteFile(testFilePath, []byte(originalContent), 0644); err != nil {
		t.Fatal(err)
	}

	patch := config.PatchConfig{
		Name:        "buildvcs_test",
		TargetFile:  "test.go",
		Description: "Test patch for buildvcs",
		Find:        `BuildBuildvcs      = "auto"`,
		Replace:     `BuildBuildvcs      = "false"`,
	}

	err = ApplyPatch(testFilePath, patch)
	if err == nil {
		t.Error("Expected error when patch target not found, got nil")
	}

	if !strings.Contains(err.Error(), "could not find target string") {
		t.Errorf("Expected 'could not find target string' error, got: %v", err)
	}
}


