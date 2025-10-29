package versions

import (
	"testing"

	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions/config"
)

func TestGetVersionConfig_MinorVersionMatch(t *testing.T) {
	cfg, err := GetVersionConfig("1.23.0")
	if err != nil {
		t.Fatalf("Expected match for 1.23.0, got error: %v", err)
	}
	if cfg.Go != "1.23" {
		t.Errorf("Expected Go version 1.23, got %s", cfg.Go)
	}
	if cfg.BaseVersion != "1.23.0" {
		t.Errorf("Expected BaseVersion 1.23.0, got %s", cfg.BaseVersion)
	}
}

func TestGetVersionConfig_PatchVersions(t *testing.T) {
	tests := []struct {
		version string
		wantGo  string
	}{
		{"1.23.1", "1.23"},
		{"1.23.5", "1.23"},
		{"1.23.99", "1.23"},
		{"1.19.0", "1.19"},
		{"1.19.13", "1.19"},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			cfg, err := GetVersionConfig(tt.version)
			if err != nil {
				t.Fatalf("Expected match for %s, got error: %v", tt.version, err)
			}
			if cfg.Go != tt.wantGo {
				t.Errorf("Expected Go version %s, got %s", tt.wantGo, cfg.Go)
			}
		})
	}
}

func TestGetVersionConfig_UnsupportedMinor(t *testing.T) {
	_, err := GetVersionConfig("1.25.0")
	if err == nil {
		t.Fatal("Expected error for unsupported version 1.25.0")
	}

	vErr, ok := err.(*VersionNotFoundError)
	if !ok {
		t.Fatalf("Expected VersionNotFoundError, got %T", err)
	}
	if vErr.Version != "1.25.0" {
		t.Errorf("Expected version 1.25.0 in error, got %s", vErr.Version)
	}
}

func TestGetVersionConfig_WithOverride(t *testing.T) {
	testConfig := config.VersionConfig{
		Go:          "1.99",
		BaseVersion: "1.99.0",
		Notes:       "Test config with overrides",
		Injections: []config.InjectionConfig{
			{
				Name:       "test_injection",
				TargetFile: "test.go",
				Line:       100,
			},
			{
				Name:       "another_injection",
				TargetFile: "test.go",
				Line:       200,
			},
		},
		Overrides: map[string]config.VersionOverride{
			"1.99.5": {
				Injections: []config.InjectionOverride{
					{
						Name: "test_injection",
						Line: 105,
					},
				},
			},
		},
	}

	SupportedVersions["1.99"] = testConfig
	defer delete(SupportedVersions, "1.99")

	t.Run("base version uses original line numbers", func(t *testing.T) {
		cfg, err := GetVersionConfig("1.99.0")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(cfg.Injections) != 2 {
			t.Fatalf("Expected 2 injections, got %d", len(cfg.Injections))
		}

		if cfg.Injections[0].Line != 100 {
			t.Errorf("Expected base line 100, got %d", cfg.Injections[0].Line)
		}
		if cfg.Injections[1].Line != 200 {
			t.Errorf("Expected base line 200, got %d", cfg.Injections[1].Line)
		}
	})

	t.Run("override version uses modified line number", func(t *testing.T) {
		cfg, err := GetVersionConfig("1.99.5")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(cfg.Injections) != 2 {
			t.Fatalf("Expected 2 injections, got %d", len(cfg.Injections))
		}

		if cfg.Injections[0].Line != 105 {
			t.Errorf("Expected overridden line 105, got %d", cfg.Injections[0].Line)
		}
		if cfg.Injections[1].Line != 200 {
			t.Errorf("Expected unchanged line 200, got %d", cfg.Injections[1].Line)
		}
	})

	t.Run("non-override version uses base config", func(t *testing.T) {
		cfg, err := GetVersionConfig("1.99.3")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if cfg.Injections[0].Line != 100 {
			t.Errorf("Expected base line 100, got %d", cfg.Injections[0].Line)
		}
	})
}

func TestGetMinorVersion(t *testing.T) {
	tests := []struct {
		version string
		want    string
		wantErr bool
	}{
		{"1.23.0", "1.23", false},
		{"1.23.5", "1.23", false},
		{"1.19.13", "1.19", false},
		{"2.0.1", "2.0", false},
		{"1", "", true},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got, err := getMinorVersion(tt.version)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for version %s", tt.version)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Expected %s, got %s", tt.want, got)
			}
		})
	}
}
