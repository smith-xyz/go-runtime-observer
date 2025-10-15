package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
	"path/filepath"

	reflectinst "github.com/smith-xyz/go-runtime-observer/instrumenter/reflect"
	"golang.org/x/tools/go/ast/astutil"
)

func main() {
	goVersion := flag.String("version", "1.21.0", "Go version to instrument")
	goSrcPath := flag.String("src", "", "Path to Go source (will download if not provided)")
	outputPath := flag.String("output", "./instrumented", "Output path for instrumented source")
	flag.Parse()

	log.Printf("Instrumenting Go %s", *goVersion)

	// TODO: Download Go source if not provided
	if *goSrcPath == "" {
		log.Fatal("Go source path required (auto-download coming soon)")
	}

	if err := instrumentReflect(*goSrcPath, *outputPath); err != nil {
		log.Fatalf("Failed to instrument reflect package: %v", err)
	}

	log.Println("Instrumentation complete!")
}

func instrumentReflect(srcPath, outputPath string) error {
	valuePath := filepath.Join(srcPath, "src", "reflect", "value.go")
	
	log.Printf("Parsing %s", valuePath)
	
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, valuePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse: %w", err)
	}

	astutil.AddImport(fset, node, "runtime/instrumentlog")

	modified := false
	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			switch fn.Name.Name {
			case "ValueOf":
				log.Println("  -> Instrumenting Reflect.ValueOf")
				reflectinst.InjectValueOf(fn)
				modified = true
			case "Call":
				log.Println("  -> Instrumenting Value.Call")
				reflectinst.InjectCall(fn)
				modified = true
			case "Method":
				log.Println("  -> Instrumenting Value.Method")
				reflectinst.InjectMethod(fn)
				modified = true
			}
		}
		return true
	})

	if !modified {
		return fmt.Errorf("no functions were instrumented")
	}

	// todo: conventionalize this so it just follows the same directory structure as Go
	outDir := filepath.Join(outputPath, "reflect")
	outFile := filepath.Join(outDir, "value.go")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	f, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer f.Close()

	log.Printf("Writing instrumented source to %s", outFile)
	return printer.Fprint(f, fset, node)
}
