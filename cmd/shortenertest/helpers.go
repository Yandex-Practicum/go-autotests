package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"golang.org/x/tools/go/ast/astutil"

	"github.com/Yandex-Practicum/go-autotests/internal/random"
)

// generateTestURL returns valid random URL
func generateTestURL(t *testing.T) string {
	t.Helper()
	return random.URL().String()
}

// importsKnownPackage checks if given file imports
func importsKnownPackage(t *testing.T, filepath string, knownPackages []string) (*ast.ImportSpec, error) {
	t.Helper()

	fset := token.NewFileSet()
	sf, err := parser.ParseFile(fset, filepath, nil, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("cannot parse file: %w", err)
	}

	importSpecs := astutil.Imports(fset, sf)
	for _, paragraph := range importSpecs {
		for _, importSpec := range paragraph {
			for _, knownImport := range knownPackages {
				if strings.Contains(importSpec.Path.Value, knownImport) {
					return importSpec, nil
				}
			}
		}
	}

	return nil, nil
}
