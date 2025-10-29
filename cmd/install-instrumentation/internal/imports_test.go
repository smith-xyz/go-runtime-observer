package internal

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestAddImport(t *testing.T) {
	testCode := `package load

import (
	"fmt"
	"os"
)

func test() {
	fmt.Println("hello")
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", testCode, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test code: %v", err)
	}

	err = AddImport(file)
	if err != nil {
		t.Fatalf("AddImport failed: %v", err)
	}

	if !HasImport(file) {
		t.Error("Import was not added")
	}

	err = AddImport(file)
	if err != nil {
		t.Fatalf("AddImport failed on second call: %v", err)
	}

	if !HasImport(file) {
		t.Error("Import was lost on second call")
	}
}

func TestHasImport(t *testing.T) {
	codeWithImport := `package load

import (
	"fmt"
	` + IMPORT_VALUE + `
)
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", codeWithImport, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse code with import: %v", err)
	}

	if !HasImport(file) {
		t.Error("Should detect existing import")
	}

	codeWithoutImport := `package load

import "fmt"
`

	file, err = parser.ParseFile(fset, "", codeWithoutImport, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse code without import: %v", err)
	}

	if HasImport(file) {
		t.Error("Should not detect import when not present")
	}
}
