package preprocessor

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	tempDirBase    string
	tempDirOnce    sync.Once
	tempDirInitErr error
	tempDirMutex   sync.RWMutex
	pathCache      = make(map[string]string)
)

func GetModuleType(filePath string, registry *Registry) string {
	if registry == nil {
		return "user"
	}

	if registry.IsStdLib(filePath) {
		return "stdlib"
	}

	if registry.IsDependencyPackage(filePath) {
		return "dependency"
	}

	return "user"
}

func EnsureTempDir() (string, error) {
	tempDirOnce.Do(func() {
		tempDirBase, tempDirInitErr = os.MkdirTemp("", "go-runtime-observer-")
	})
	return tempDirBase, tempDirInitErr
}

func EnsureModuleTypeDir(moduleType string) (string, error) {
	baseDir, err := EnsureTempDir()
	if err != nil {
		return "", err
	}

	moduleDir := filepath.Join(baseDir, moduleType)
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create module type directory: %w", err)
	}

	return moduleDir, nil
}

func GetInstrumentedFilePath(originalPath string, registry *Registry) (string, error) {
	tempDirMutex.RLock()
	if cached, ok := pathCache[originalPath]; ok {
		tempDirMutex.RUnlock()
		return cached, nil
	}
	tempDirMutex.RUnlock()

	baseDir, err := EnsureTempDir()
	if err != nil {
		return "", err
	}

	absPath, err := filepath.Abs(originalPath)
	if err != nil {
		absPath = originalPath
	}

	tempPath := filepath.Join(baseDir, absPath)

	if err := os.MkdirAll(filepath.Dir(tempPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	tempDirMutex.Lock()
	pathCache[originalPath] = tempPath
	tempDirMutex.Unlock()

	return tempPath, nil
}

func CleanupTempDir() error {
	tempDirMutex.Lock()
	defer tempDirMutex.Unlock()

	if tempDirBase == "" {
		return nil
	}

	err := os.RemoveAll(tempDirBase)

	tempDirBase = ""
	pathCache = make(map[string]string)
	tempDirOnce = sync.Once{}

	return err
}
