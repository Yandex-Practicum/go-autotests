package main

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

var (
	testFileFound = errors.New("test file found")
)

// TestIteration2 checks that students code contains test files
func TestIteration2(t *testing.T) {
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

		if strings.HasSuffix(d.Name(), "_test.go") {
			return testFileFound
		}

		return nil
	})

	if errors.Is(err, testFileFound) {
		return
	}

	if err == nil {
		t.Error("No test files have been found")
		return
	}

	t.Errorf("unexpected error: %s", err)
}