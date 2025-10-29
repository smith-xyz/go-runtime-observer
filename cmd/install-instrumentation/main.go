package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal"
	"github.com/smith-xyz/go-runtime-observer/cmd/install-instrumentation/internal/versions"
)

func main() {
	goVersion := flag.String("go-version", "1.23.0", "Go version to instrument for")
	listVersions := flag.Bool("list-versions", false, "List all supported Go versions and exit")
	flag.Parse()

	if *listVersions {
		fmt.Println("Supported Go versions:")
		for _, v := range versions.ListSupportedVersions() {
			config, _ := versions.GetVersionConfig(v)
			fmt.Printf("  - %s (%s)\n", v, config.Notes)
		}
		os.Exit(0)
	}

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <path-to-go-source-root>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	goSourceRoot := flag.Arg(0)

	versionConfig, err := versions.GetVersionConfig(*goVersion)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		supported := versions.ListSupportedVersions()
		fmt.Fprintf(os.Stderr, "Supported Go versions:\n")
		for _, v := range supported {
			fmt.Fprintf(os.Stderr, "  - %s\n", v)
		}
		fmt.Fprintf(os.Stderr, "\nTo add support for Go %s, update cmd/install-instrumentation/internal/versions.go\n", *goVersion)
		os.Exit(1)
	}

	fmt.Printf("Instrumenting Go %s\n", versionConfig.Go)

	if len(versionConfig.Injections) > 0 {
		fmt.Printf("Injections (%d):\n", len(versionConfig.Injections))
		for _, inj := range versionConfig.Injections {
			fmt.Printf("  - %s: line %d (%s)\n", inj.Name, inj.Line, inj.Description)
		}
	}

	if len(versionConfig.Patches) > 0 {
		fmt.Printf("Patches (%d):\n", len(versionConfig.Patches))
		for _, patch := range versionConfig.Patches {
			fmt.Printf("  - %s: %s\n", patch.Name, patch.Description)
		}
	}

	if err := internal.InstrumentFile(goSourceRoot, *goVersion); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Successfully instrumented Go toolchain")
}
