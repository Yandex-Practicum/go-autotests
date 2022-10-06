package main

// Basic imports
import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/stretchr/testify/suite"
)

// Iteration17Suite является сьютом с тестами и состоянием для инкремента
type Iteration17Suite struct {
	suite.Suite
}

// SetupSuite подготавливает необходимые зависимости
func (suite *Iteration17Suite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")
}

// TestDocsComments пробует проверить налиция документационных комментариев в коде
func (suite *Iteration17Suite) TestDocsComments() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "doc", "-all", "-short", flagTargetSourcePath)
	cmd.Env = os.Environ() // pass parent envs
	out, err := cmd.CombinedOutput()

	suite.NoErrorf(err, "Невозможно получить результат выполнения команды: %s. Ошибка: %w", cmd, err)
	suite.Emptyf(out, "Не найдена документация проекта:\n\n%s", cmd)
}

// TestExamplePresence пробует рекурсивно найти хотя бы один файл example_test.go в директории с исходным кодом проекта
func (suite *Iteration17Suite) TestExamplePresence() {
	err := filepath.WalkDir(flagTargetSourcePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			// пропускаем служебные директории
			if d.Name() == "vendor" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			// проваливаемся в директорию
			return nil
		}

		// проверяем имя файла
		if strings.HasSuffix(d.Name(), "example_test.go") {
			// возвращаем сигнальную ошибку
			return errUsageFound
		}

		return nil
	})

	// проверяем сигнальную ошибку
	if errors.Is(err, errUsageFound) {
		// найден хотя бы один файл
		return
	}

	if err == nil {
		suite.T().Errorf("Не найден ни один файл example_test.go по пути %s", flagTargetSourcePath)
		return
	}
	suite.T().Errorf("Неожиданная ошибка при поиске файла example_test.go по пути %s: %s", flagTargetSourcePath, err)
}
