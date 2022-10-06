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

// Iteration16Suite является сьютом с тестами и состоянием для инкремента
type Iteration16Suite struct {
	suite.Suite
}

// SetupSuite подготавливает необходимые зависимости
func (suite *Iteration16Suite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")
}

// TestStylingDiff пробует проверить правильность форматирования кода в проекте
func (suite *Iteration16Suite) TestStylingDiff() {
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
