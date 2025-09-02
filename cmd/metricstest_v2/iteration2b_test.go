package main

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/stretchr/testify/suite"
)

type Iteration2BSuite struct {
	suite.Suite

	coverRegex *regexp.Regexp
}

// SetupSuite подготавливает необходимые зависимости
func (suite *Iteration2BSuite) SetupSuite() {
	// проверяем наличие необходимых флагов
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")

	// подготавливаем регулярное выражение для проверки результатов тестов
	regex, err := regexp.Compile(`coverage: (\d+.\d)% of statements`)
	suite.Require().NoError(err)

	suite.coverRegex = regex
}

// TestFilesPresence пробует рекурсивно найти хотя бы один тестовый файл в директории с исходным кодом проекта
func (suite *Iteration2BSuite) TestFilesPresence() {
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

		// проверяем суффикс имени файла
		if strings.HasSuffix(d.Name(), "_test.go") {
			// возвращаем сигнальную ошибку
			return errUsageFound
		}

		return nil
	})

	// проверяем сигнальную ошибку
	if errors.Is(err, errUsageFound) {
		// найден хотя бы один файл с тестами
		return
	}

	if err == nil {
		suite.T().Errorf("Не найден ни один тестовый файл по пути %s", flagTargetSourcePath)
		return
	}
	suite.T().Errorf("Неожиданная ошибка при поиске тестовых файлов по пути %s: %s", flagTargetSourcePath, err)
}

// TestServerCoverage пытается получить и прочитать результаты выполнения команды go test -cover
func (suite *Iteration2BSuite) TestServerCoverage() {
	// подготавливаем сроку пути до исходного кода проекта
	sourcePath := strings.TrimRight(flagTargetSourcePath, "/") + "/..."

	// будем ожидать завершения выполнения команды не более 2 минут
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// запускаем тесты
	cmd := exec.CommandContext(ctx, "go", "test", "-cover", sourcePath)
	cmd.Env = os.Environ() // pass parent envs
	// получаем вывод команды
	out, err := cmd.CombinedOutput()
	suite.Assert().NoError(err, "Невозможно получить результат выполнения команды: %s. Вывод:\n\n %s", cmd, out)

	// проверяем вывод с помощью регулярного выражения
	matched := suite.coverRegex.Match(out)
	found := suite.Assert().True(matched, "Отсутствует информация о покрытии кода тестами, команда: %s", cmd)

	if !found {
		suite.T().Logf("Вывод команды:\n\n%s", string(out))
	}
}
