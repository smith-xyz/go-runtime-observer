package internal

import (
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions"
	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions/config"
)

func InstrumentFile(goSourceRoot string, goVersion string) error {

	// Algorithm for working with files: code injection bottom to top of file -> patches match to text -> add any necessary imports
	// This ensures our record of line numbers are accurate regardless of line shifts

	versionConfig, err := versions.GetVersionConfig(goVersion)
	if err != nil {
		return fmt.Errorf("version config error: %w", err)
	}

	// Group injections by file
	fileInjections := make(map[string][]config.InjectionConfig)
	for _, injection := range versionConfig.Injections {
		filePath := filepath.Join(goSourceRoot, injection.TargetFile)
		fileInjections[filePath] = append(fileInjections[filePath], injection)
	}

	// Step 1: Process injections (bottom to top per file)
	for filePath, injections := range fileInjections {
		// Sort injections for this file in descending order by line
		sort.Slice(injections, func(i, j int) bool {
			return injections[i].Line > injections[j].Line
		})

		// Process injections from bottom to top
		for _, injection := range injections {
			if err := InjectCode(filePath, injection); err != nil {
				return fmt.Errorf("failed to inject %s: %w", injection.Name, err)
			}
		}
	}

	// Step 2: Apply patches (text-based, immune to line shifts)
	for _, patch := range versionConfig.Patches {
		filePath := filepath.Join(goSourceRoot, patch.TargetFile)
		if err := ApplyPatch(filePath, patch); err != nil {
			return fmt.Errorf("failed to apply patch %s: %w", patch.Name, err)
		}
	}

	// Step 3: Add imports last (AST-based, handles line shifts automatically)
	processedFiles := make(map[string]bool)
	for filePath := range fileInjections {
		if !processedFiles[filePath] {
			content, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", filePath, err)
			}

			if !strings.Contains(string(content), IMPORT_VALUE) {
				fset := token.NewFileSet()
				file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
				if err != nil {
					return fmt.Errorf("failed to parse %s: %w", filePath, err)
				}

				if err := AddImport(file); err != nil {
					return fmt.Errorf("failed to add import to %s: %w", filePath, err)
				}

				output, err := os.Create(filePath)
				if err != nil {
					return fmt.Errorf("failed to create output file %s: %w", filePath, err)
				}

				if err := format.Node(output, fset, file); err != nil {
					output.Close()
					return fmt.Errorf("failed to format %s: %w", filePath, err)
				}
				output.Close()
			}

			processedFiles[filePath] = true
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
