package internal

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"strings"

	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions/config"
)

const (
	PREPROCESSOR_PKG = "preprocessor"
)

func InjectCode(filePath string, injection config.InjectionConfig) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, filePath, content, parser.ParseComments); err != nil {
		return fmt.Errorf("original file is not valid Go: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	if injection.Line < 1 || injection.Line > len(lines) {
		return fmt.Errorf("line %d is out of range (file has %d lines)",
			injection.Line, len(lines))
	}

	codeLines, err := generateInjectionCode(injection)
	if err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}

	lines = injectCodeAtLine(lines, injection.Line, codeLines)

	modifiedContent := strings.Join(lines, "\n")

	if _, err := parser.ParseFile(token.NewFileSet(), filePath, modifiedContent, parser.ParseComments); err != nil {
		return fmt.Errorf("generated code is not valid Go: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(modifiedContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func generateInjectionCode(injection config.InjectionConfig) ([]string, error) {
	if err := validateInjectionConfig(injection); err != nil {
		return nil, err
	}

	var lines []string

	if len(injection.Instrument.Result) > 0 {
		instrumentLine := fmt.Sprintf("\t%s = %s.%s(%s)",
			strings.Join(injection.Instrument.Result, ", "),
			PREPROCESSOR_PKG,
			injection.Instrument.Function,
			strings.Join(injection.Instrument.Args, ", "))
		lines = append(lines, instrumentLine)
	} else {
		instrumentLine := fmt.Sprintf("\t%s.%s(%s)",
			PREPROCESSOR_PKG,
			injection.Instrument.Function,
			strings.Join(injection.Instrument.Args, ", "))
		lines = append(lines, instrumentLine)
	}

	reparseLine := fmt.Sprintf("\t%s = %s(%s)",
		strings.Join(injection.Reparse.Result, ", "),
		injection.Reparse.Function,
		strings.Join(injection.Reparse.Args, ", "))
	lines = append(lines, reparseLine)

	return lines, nil
}

// Important: anything that appears in config
func validateInjectionConfig(injection config.InjectionConfig) error {
	if injection.Instrument.Function != ALLOWED_INSTRUMENT_FUNCTION {
		return fmt.Errorf("instrument function must be %q, got %q",
			ALLOWED_INSTRUMENT_FUNCTION, injection.Instrument.Function)
	}

	if len(injection.Instrument.Args) != REQUIRED_INSTRUMENT_ARG_COUNT {
		return fmt.Errorf("instrument function requires exactly %d arguments, got %d",
			REQUIRED_INSTRUMENT_ARG_COUNT, len(injection.Instrument.Args))
	}

	if len(injection.Instrument.Result) > 0 && len(injection.Instrument.Result) != REQUIRED_INSTRUMENT_RESULT_COUNT {
		return fmt.Errorf("instrument function must return exactly %d results or none, got %d",
			REQUIRED_INSTRUMENT_RESULT_COUNT, len(injection.Instrument.Result))
	}

	if len(injection.Reparse.Result) != REQUIRED_REPARSE_RESULT_COUNT {
		return fmt.Errorf("reparse must assign to exactly %d results, got %d",
			REQUIRED_REPARSE_RESULT_COUNT, len(injection.Reparse.Result))
	}

	if !isAllowedReparseFunction(injection.Reparse.Function) {
		return fmt.Errorf("reparse function must be one of %v, got %q",
			AllowedReparseFunctions, injection.Reparse.Function)
	}

	return nil
}

func injectCodeAtLine(lines []string, lineNum int, codeLines []string) []string {
	idx := lineNum - 1

	result := make([]string, 0, len(lines)+len(codeLines))
	result = append(result, lines[:idx+1]...)
	result = append(result, codeLines...)
	result = append(result, lines[idx+1:]...)

	return result
}
