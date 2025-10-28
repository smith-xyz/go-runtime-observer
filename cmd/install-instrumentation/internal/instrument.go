package internal

import (
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions"
	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions/config"
)

func InstrumentFile(goSourceRoot string, goVersion string) error {
	versionConfig, err := versions.GetVersionConfig(goVersion)
	if err != nil {
		return fmt.Errorf("version config error: %w", err)
	}

	processedFiles := make(map[string]bool)

	for _, injection := range versionConfig.Injections {
		filePath := filepath.Join(goSourceRoot, injection.TargetFile)
		
		if !processedFiles[filePath] {
			content, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", injection.TargetFile, err)
			}

			if !strings.Contains(string(content), IMPORT_VALUE) {
				fset := token.NewFileSet()
				file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
				if err != nil {
					return fmt.Errorf("failed to parse %s: %w", injection.TargetFile, err)
				}

				if err := AddImport(file); err != nil {
					return fmt.Errorf("failed to add import to %s: %w", injection.TargetFile, err)
				}

				output, err := os.Create(filePath)
				if err != nil {
					return fmt.Errorf("failed to create output file %s: %w", injection.TargetFile, err)
				}

				if err := format.Node(output, fset, file); err != nil {
					output.Close()
					return fmt.Errorf("failed to format %s: %w", injection.TargetFile, err)
				}
				output.Close()
			}

			processedFiles[filePath] = true
		}

		if err := InjectCode(filePath, injection); err != nil {
			return fmt.Errorf("failed to inject %s: %w", injection.Name, err)
		}
	}

	for _, patch := range versionConfig.Patches {
		filePath := filepath.Join(goSourceRoot, patch.TargetFile)
		if err := ApplyPatch(filePath, patch); err != nil {
			return fmt.Errorf("failed to apply patch %s: %w", patch.Name, err)
		}
	}

	return nil
}

func ApplyPatch(filePath string, patch config.PatchConfig) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", patch.TargetFile, err)
	}

	if !strings.Contains(string(content), patch.Find) {
		if strings.Contains(string(content), patch.Replace) {
			return nil
		}
		return fmt.Errorf("could not find target string in %s", patch.TargetFile)
	}

	newContent := strings.Replace(string(content), patch.Find, patch.Replace, 1)
	
	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", patch.TargetFile, err)
	}

	return nil
}

