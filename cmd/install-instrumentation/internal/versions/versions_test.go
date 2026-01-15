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
	_, err := GetVersionConfig("1.0.0")
	if err == nil {
		t.Fatal("Expected error for unsupported version 1.0.0")
	}

	vErr, ok := err.(*VersionNotFoundError)
	if !ok {
		t.Fatalf("Expected VersionNotFoundError, got %T", err)
	}
	if vErr.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0 in error, got %s", vErr.Version)
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

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		v1   string
		v2   string
		want int
	}{
		{"1.23.0", "1.23.0", 0},
		{"1.23.1", "1.23.0", 1},
		{"1.23.0", "1.23.1", -1},
		{"1.22.0", "1.23.0", -1},
		{"1.23.0", "1.22.0", 1},
		{"1.19.13", "1.19.12", 1},
		{"1.19.12", "1.19.13", -1},
		{"1.24.3", "1.24.3", 0},
		{"1.24.3", "1.24.9", -1},
		{"1.24.9", "1.24.3", 1},
		{"2.0.0", "1.99.99", 1},
		{"1.99.99", "2.0.0", -1},
		{"1.24.3", "1.25.0", -1},
		{"1.24.3", "1.24.3", 0},
	}

	for _, tt := range tests {
		t.Run(tt.v1+"_vs_"+tt.v2, func(t *testing.T) {
			got := compareVersions(tt.v1, tt.v2)
			if got != tt.want {
				t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.v1, tt.v2, got, tt.want)
			}

			// Test symmetry: comparing v2 vs v1 should give opposite result
			if tt.want != 0 {
				wantReverse := -tt.want
				gotReverse := compareVersions(tt.v2, tt.v1)
				if gotReverse != wantReverse {
					t.Errorf("compareVersions(%q, %q) = %d, want %d (reverse of %d)", tt.v2, tt.v1, gotReverse, wantReverse, tt.want)
				}
			}
		})
	}
}

func TestApplyOverrides_Cascading(t *testing.T) {
	testConfig := config.VersionConfig{
		Go:          "1.99",
		BaseVersion: "1.99.0",
		Notes:       "Test config with cascading overrides",
		Injections: []config.InjectionConfig{
			{
				Name:       "test_injection",
				TargetFile: "test.go",
				Line:       100,
			},
		},
		Overrides: map[string]config.VersionOverride{
			"1.99.3": {
				Injections: []config.InjectionOverride{
					{
						Name: "test_injection",
						Line: 105,
					},
				},
			},
		},
	}

	tests := []struct {
		version  string
		wantLine int
		desc     string
	}{
		{"1.99.0", 100, "base version before override"},
		{"1.99.2", 100, "patch before override"},
		{"1.99.3", 105, "exact override version"},
		{"1.99.4", 105, "patch after override (cascades)"},
		{"1.99.9", 105, "later patch (cascades)"},
		{"1.99.99", 105, "much later patch (cascades)"},
	}

	for _, tt := range tests {
		t.Run(tt.desc+"_"+tt.version, func(t *testing.T) {
			result := applyOverrides(testConfig, tt.version)
			if len(result.Injections) != 1 {
				t.Fatalf("Expected 1 injection, got %d", len(result.Injections))
			}
			if result.Injections[0].Line != tt.wantLine {
				t.Errorf("Expected line %d for version %s, got %d", tt.wantLine, tt.version, result.Injections[0].Line)
			}
		})
	}
}

func TestApplyOverrides_MultipleOverridesPicksLatest(t *testing.T) {
	testConfig := config.VersionConfig{
		Go:          "1.99",
		BaseVersion: "1.99.0",
		Injections: []config.InjectionConfig{
			{
				Name:       "test_injection",
				TargetFile: "test.go",
				Line:       100,
			},
		},
		Overrides: map[string]config.VersionOverride{
			"1.99.3": {
				Injections: []config.InjectionOverride{
					{Name: "test_injection", Line: 105},
				},
			},
			"1.99.5": {
				Injections: []config.InjectionOverride{
					{Name: "test_injection", Line: 110},
				},
			},
		},
	}

	tests := []struct {
		version  string
		wantLine int
		desc     string
	}{
		{"1.99.0", 100, "before any override"},
		{"1.99.2", 100, "before first override"},
		{"1.99.3", 105, "exact first override"},
		{"1.99.4", 105, "between overrides (uses first)"},
		{"1.99.5", 110, "exact second override"},
		{"1.99.6", 110, "after second override (uses latest)"},
		{"1.99.9", 110, "much later (uses latest)"},
	}

	for _, tt := range tests {
		t.Run(tt.desc+"_"+tt.version, func(t *testing.T) {
			result := applyOverrides(testConfig, tt.version)
			if len(result.Injections) != 1 {
				t.Fatalf("Expected 1 injection, got %d", len(result.Injections))
			}
			if result.Injections[0].Line != tt.wantLine {
				t.Errorf("Expected line %d for version %s, got %d", tt.wantLine, tt.version, result.Injections[0].Line)
			}
		})
	}
}
