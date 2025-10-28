package v1_19

import (
	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions/config"
)

func GetConfig() config.VersionConfig {
	return config.VersionConfig{
		Go:    "1.19",
		Notes: "Initial stable implementation - pkg.go with Happy: label and goFilesPackage",
		Injections: []config.InjectionConfig{
			{
				Name:        "dependency",
				TargetFile:  "src/cmd/go/internal/load/pkg.go",
				Line:        897,
				Description: "Injects after Happy: label in dependency resolution path",
				Instrument: config.InstrumentCall{
					Function: "InstrumentPackageFiles",
					Args:     []string{"data.p.GoFiles", "data.p.Dir"},
					Result:   []string{"data.p.GoFiles", "data.p.Dir"},
				},
				Reparse: config.ReparseCall{
					Result:   []string{"data.p", "data.err"},
					Function: "cfg.BuildContext.ImportDir",
					Args:     []string{"data.p.Dir", "buildMode"},
				},
			},
			{
				Name:        "command_line",
				TargetFile:  "src/cmd/go/internal/load/pkg.go",
				Line:        3029,
				Description: "Injects after ImportDir call in goFilesPackage for command-line files",
				Instrument: config.InstrumentCall{
					Function: "InstrumentPackageFiles",
					Args:     []string{"bp.GoFiles", "dir"},
					Result:   []string{"bp.GoFiles", "dir"},
				},
				Reparse: config.ReparseCall{
					Result:   []string{"bp", "err"},
					Function: "ctxt.ImportDir",
					Args:     []string{"dir", "0"},
				},
			},
		},
		Patches: []config.PatchConfig{
			{
				Name:        "buildvcs_default",
				TargetFile:  "src/cmd/go/internal/cfg/cfg.go",
				Description: "Disable VCS stamping by default to support temp directory instrumentation",
				Find:        `BuildBuildvcs          = "auto"`,
				Replace:     `BuildBuildvcs          = "false"`,
			},
		},
	}
}

