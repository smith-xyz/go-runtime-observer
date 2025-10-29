package versions

import (
	"fmt"
	"strings"

	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions/config"
	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions/v1_19"
	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions/v1_20"
	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions/v1_21"
	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions/v1_22"
	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions/v1_23"
	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions/v1_24"
)

// SupportedVersions maps Go minor versions (e.g., "1.23") to their base instrumentation configurations.
// Each version's config is defined in a separate versions/vX_Y/ directory for maintainability.
// Patch-specific overrides are defined within each config's Overrides map.
// To add a new Go version:
//  1. Create versions/vX_Y/config.go with GetConfig() function
//  2. Import the package here
//  3. Add the minor version to this map
var SupportedVersions = map[string]config.VersionConfig{
	"1.19": v1_19.GetConfig(),
	"1.20": v1_20.GetConfig(),
	"1.21": v1_21.GetConfig(),
	"1.22": v1_22.GetConfig(),
	"1.23": v1_23.GetConfig(),
	"1.24": v1_24.GetConfig(),
}

func GetVersionConfig(version string) (*config.VersionConfig, error) {
	minorVersion, err := getMinorVersion(version)
	if err != nil {
		return nil, err
	}

	baseConfig, exists := SupportedVersions[minorVersion]
	if !exists {
		return nil, &VersionNotFoundError{
			Version:   version,
			BestMatch: "",
		}
	}

	finalConfig := applyOverrides(baseConfig, version)
	return &finalConfig, nil
}

func getMinorVersion(version string) (string, error) {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid version format: %s (expected at least major.minor)", version)
	}
	return fmt.Sprintf("%s.%s", parts[0], parts[1]), nil
}

func applyOverrides(baseConfig config.VersionConfig, targetVersion string) config.VersionConfig {
	override, exists := baseConfig.Overrides[targetVersion]
	if !exists {
		return baseConfig
	}

	result := baseConfig

	if len(override.Injections) > 0 {
		result.Injections = make([]config.InjectionConfig, len(baseConfig.Injections))
		copy(result.Injections, baseConfig.Injections)

		for _, overrideInj := range override.Injections {
			for i := range result.Injections {
				if result.Injections[i].Name == overrideInj.Name {
					result.Injections[i].Line = overrideInj.Line
					break
				}
			}
		}
	}

	if len(override.Patches) > 0 {
		result.Patches = override.Patches
	}

	return result
}

func ListSupportedVersions() []string {
	versions := make([]string, 0, len(SupportedVersions))
	for v := range SupportedVersions {
		versions = append(versions, v)
	}
	return versions
}

type VersionNotFoundError struct {
	Version   string
	BestMatch string
}

func (e *VersionNotFoundError) Error() string {
	if e.BestMatch != "" {
		return fmt.Sprintf("no exact configuration found for Go version %s, attempted fallback to %s but it's incompatible", e.Version, e.BestMatch)
	}
	return "no configuration found for Go version " + e.Version
}
