package versions

import (
	"testing"

	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions/config"
)

func TestGetVersionConfig_ExactMatch(t *testing.T) {
	cfg, err := GetVersionConfig("1.23.0")
	if err != nil {
		t.Fatalf("Expected exact match for 1.23.0, got error: %v", err)
	}
	if cfg.Go != "1.23.0" {
		t.Errorf("Expected Go version 1.23.0, got %s", cfg.Go)
	}
}

func TestGetVersionConfig_PatchFallback(t *testing.T) {
	cfg, err := GetVersionConfig("1.23.1")
	if err != nil {
		t.Fatalf("Expected fallback to 1.23.0 for 1.23.1, got error: %v", err)
	}
	if cfg.Go != "1.23.0" {
		t.Errorf("Expected fallback to 1.23.0, got %s", cfg.Go)
	}

	cfg, err = GetVersionConfig("1.23.5")
	if err != nil {
		t.Fatalf("Expected fallback to 1.23.0 for 1.23.5, got error: %v", err)
	}
	if cfg.Go != "1.23.0" {
		t.Errorf("Expected fallback to 1.23.0, got %s", cfg.Go)
	}
}

func TestGetVersionConfig_NoMatch(t *testing.T) {
	_, err := GetVersionConfig("1.24.0")
	if err == nil {
		t.Fatal("Expected error for unsupported version 1.24.0")
	}

	vErr, ok := err.(*VersionNotFoundError)
	if !ok {
		t.Fatalf("Expected VersionNotFoundError, got %T", err)
	}
	if vErr.Version != "1.24.0" {
		t.Errorf("Expected version 1.24.0 in error, got %s", vErr.Version)
	}
}

func TestGetVersionConfig_LowerPatchNotAllowed(t *testing.T) {
	SupportedVersions["1.22.3"] = config.VersionConfig{
		Go:    "1.22.3",
		Notes: "Test version",
	}
	defer delete(SupportedVersions, "1.22.3")

	_, err := GetVersionConfig("1.22.2")
	if err == nil {
		t.Fatal("Expected error when requesting 1.22.2 with only 1.22.3 available")
	}
}

func TestGetVersionConfig_HighestPatchSelected(t *testing.T) {
	SupportedVersions["1.22.0"] = config.VersionConfig{
		Go:    "1.22.0",
		Notes: "Test version 1.22.0",
	}
	SupportedVersions["1.22.2"] = config.VersionConfig{
		Go:    "1.22.2",
		Notes: "Test version 1.22.2",
	}
	defer func() {
		delete(SupportedVersions, "1.22.0")
		delete(SupportedVersions, "1.22.2")
	}()

	cfg, err := GetVersionConfig("1.22.5")
	if err != nil {
		t.Fatalf("Expected fallback to 1.22.2 for 1.22.5, got error: %v", err)
	}
	if cfg.Go != "1.22.2" {
		t.Errorf("Expected fallback to highest patch 1.22.2, got %s", cfg.Go)
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		version string
		major   int
		minor   int
		patch   int
		wantErr bool
	}{
		{"1.23.0", 1, 23, 0, false},
		{"1.23.5", 1, 23, 5, false},
		{"2.0.1", 2, 0, 1, false},
		{"1.23", 0, 0, 0, true},
		{"1.23.x", 0, 0, 0, true},
		{"invalid", 0, 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			major, minor, patch, err := parseVersion(tt.version)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for version %s", tt.version)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if major != tt.major || minor != tt.minor || patch != tt.patch {
				t.Errorf("Expected %d.%d.%d, got %d.%d.%d", tt.major, tt.minor, tt.patch, major, minor, patch)
			}
		})
	}
}

