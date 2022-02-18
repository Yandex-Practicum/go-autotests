package main

// Basic imports
import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/stretchr/testify/suite"
)

// Iteration17Suite is a suite of autotests
type Iteration17Suite struct {
	suite.Suite
}

// SetupSuite bootstraps suite dependencies
func (suite *Iteration17Suite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")
}

// TestStylingDiff attempts to check if source code is properly formatted via appropriate tooling
func (suite *Iteration17Suite) TestStylingDiff() {
	gofmtErr := checkGofmtStyling(flagTargetSourcePath)
	goimportsErr := checkGoimportsStyling(flagTargetSourcePath)

	if gofmtErr == nil || goimportsErr == nil {
		return
	}

	suite.Assert().NoError(gofmtErr, "Ошибка проверки форматирования с помощью gofmt")
	suite.Assert().NoError(goimportsErr, "Ошибка проверки форматирования с помощью goimports")
}

func checkGofmtStyling(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gofmt", "-l", "-s", path)
	cmd.Env = os.Environ() // pass parent envs
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Невозможно получить результат выполнения команды: %s. Ошибка: %w", cmd, err)
	}
	if len(out) > 0 {
		return fmt.Errorf("Найдены неотформатированные файлы:\n\n%s", cmd)
	}
	return nil
}

func checkGoimportsStyling(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "goimports", "-l", path)
	cmd.Env = os.Environ() // pass parent envs
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Невозможно получить результат выполнения команды: %s. Ошибка: %w", cmd, err)
	}
	if len(out) > 0 {
		return fmt.Errorf("Найдены неотформатированные файлы:\n\n%s", cmd)
	}
	return nil
}
