package versions

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions/config"
	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions/v1_19"
	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions/v1_20"
	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions/v1_21"
	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions/v1_22"
	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions/v1_23"
	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions/v1_24"
)

// SupportedVersions maps Go version strings to their instrumentation configurations.
// Each version's config is defined in a separate versions/vX_Y/ directory for maintainability.
// To add a new Go version:
//  1. Create versions/vX_Y/config.go with GetConfig() function
//  2. Import the package here
//  3. Add the version to this map
var SupportedVersions = map[string]config.VersionConfig{
	"1.19":   v1_19.GetConfig(),
	"1.20":   v1_20.GetConfig(),
	"1.21.0": v1_21.GetConfig(),
	"1.22.0": v1_22.GetConfig(),
	"1.23.0": v1_23.GetConfig(),
	"1.24.0": v1_24.GetConfig(),
}

func GetVersionConfig(version string) (*config.VersionConfig, error) {
	if cfg, exists := SupportedVersions[version]; exists {
		return &cfg, nil
	}

	bestMatch, err := findBestMatch(version)
	if err != nil {
		return nil, &VersionNotFoundError{
			Version:   version,
			BestMatch: bestMatch,
		}
	}

	cfg := SupportedVersions[bestMatch]
	return &cfg, nil
}

func findBestMatch(targetVersion string) (string, error) {
	major, minor, patch, err := parseVersion(targetVersion)
	if err != nil {
		return "", err
	}

	var bestMatch string
	var bestPatch int = -1

	for supportedVersion := range SupportedVersions {
		sMajor, sMinor, sPatch, err := parseVersion(supportedVersion)
		if err != nil {
			continue
		}

		if sMajor == major && sMinor == minor && sPatch <= patch {
			if sPatch > bestPatch {
				bestPatch = sPatch
				bestMatch = supportedVersion
			}
		}
	}

	if bestMatch == "" {
		return "", fmt.Errorf("no compatible version found for %s (major.minor %d.%d)", targetVersion, major, minor)
	}

	return bestMatch, nil
}

func parseVersion(version string) (major, minor, patch int, err error) {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid version format: %s (expected major.minor.patch)", version)
	}

	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid patch version: %s", parts[2])
	}

	return major, minor, patch, nil
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
