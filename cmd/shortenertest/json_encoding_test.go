package main

import (
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/ast/astutil"
)

var (
	knownEncodingLibs = []string{
		"encoding/json",
		"github.com/mailru/easyjson",
		"github.com/pquerna/ffjson",
	}
)

// TestUsesJSONEncoder checks that students code uses known JSON encoding library
func TestUsesJSONEncoder(t *testing.T) {
	fset := token.NewFileSet()

	err := filepath.WalkDir(config.SourceRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			// skip vendor directory
			if d.Name() == "vendor" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			// dive into regular directory
			return nil
		}

		// skip test files or non-Go files
		if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}

		sf, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("cannot parse AST of file: %s: %w", path, err)
		}

		importSpecs := astutil.Imports(fset, sf)
		if importsKnownPackage(importSpecs, knownEncodingLibs) {
			return importFound
		}

		return nil
	})

	if errors.Is(err, importFound) {
		return
	}

	if err == nil {
		t.Error("No import of known json encoding library has been found")
		return
	}

	t.Errorf("unexpected error: %s", err)
}

