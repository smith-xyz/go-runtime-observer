package preprocessor

import (
	"path/filepath"
	"strings"
)

func ProcessStdlibFile(filePath string, registry *Registry) ([]byte, bool, error) {
	if registry == nil {
		return nil, false, nil
	}

	packageName := extractStdlibPackageName(filePath)
	if packageName == "" {
		return nil, false, nil
	}

	astInstr, ok := registry.GetStdlibASTInstrumentation(packageName)
	if !ok {
		return nil, false, nil
	}

	if len(astInstr.Functions) == 0 && len(astInstr.Methods) == 0 {
		return nil, false, nil
	}

	return instrumentStdlibFile(filePath, packageName, astInstr.Functions, astInstr.Methods)
}

func extractStdlibPackageName(filePath string) string {
	parts := strings.Split(filepath.ToSlash(filePath), "/")

	for i, part := range parts {
		if part == "src" && i+1 < len(parts) {
			nextPart := parts[i+1]

			if nextPart == "runtime_observe_instrumentation" {
				return ""
			}

			if strings.Contains(nextPart, ".") {
				continue
			}

			return nextPart
		}
	}

	return ""
}
