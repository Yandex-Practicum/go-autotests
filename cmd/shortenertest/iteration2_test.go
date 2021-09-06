package main

// Basic imports
import (
	"context"
	"errors"
	"io/fs"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/stretchr/testify/suite"
)

// Iteration2Suite is a suite of autotests
type Iteration2Suite struct {
	suite.Suite

	coverRegex *regexp.Regexp
}

// SetupSuite bootstraps suite dependencies
func (suite *Iteration2Suite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path flag required")

	regex, err := regexp.Compile(`coverage: (\d+.\d)% of statements`)
	suite.Require().NoError(err)

	suite.coverRegex = regex
}

// TestFilesPresence attempts to recursively find at least one Go test file in source path
func (suite *Iteration2Suite) TestFilesPresence() {
	err := filepath.WalkDir(flagTargetSourcePath, func(path string, d fs.DirEntry, err error) error {
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
			return errUsageFound
		}

		return nil
	})

	if errors.Is(err, errUsageFound) {
		return
	}

	if err == nil {
		suite.T().Errorf("No test files have been found in %s", flagTargetSourcePath)
		return
	}
	suite.T().Errorf("unexpected error: %s", err)
}

// TestServerCoverage attempts to obtain and parse coverage report using standard Go tooling
func (suite *Iteration2Suite) TestServerCoverage() {
	sourcePath := strings.TrimRight(flagTargetSourcePath, "/") + "/..."

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "test", "-cover", sourcePath)
	out, err := cmd.CombinedOutput()
	suite.Assert().NoError(err, "got unexpected error from command: %s", cmd)

	matched := suite.coverRegex.Match(out)
	found := suite.Assert().True(matched, "no test coverage found in report of command: %s", cmd)

	if !found {
		suite.T().Logf("program output was:\n\n%s", string(out))
	}
}
