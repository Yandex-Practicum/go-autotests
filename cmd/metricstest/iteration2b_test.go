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
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

func TestIteration2B(t *testing.T) {
	commonEnv := New(t)
	table := []struct {
		name, path string
	}{
		{
			"agent",
			AgentSourcePath(commonEnv),
		},
		{
			"server",
			ServerSourcePath(commonEnv),
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			t.Run("TestFilesPresence", func(t *testing.T) {
				e := New(t)

				sourcePath := test.path

				hasTestFile := false
				err := filepath.WalkDir(sourcePath, func(path string, d fs.DirEntry, err error) error {
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
						hasTestFile = true
						return filepath.SkipAll
					}

					return nil
				})

				e.NoErrorf(err, "Неожиданная ошибка при поиске тестовых файлов по пути %s: %s", sourcePath, err)

				if !hasTestFile {
					e.Errorf("Не найден ни один тестовый файл по пути %s", sourcePath)
				}
			})

			t.Run("TestAgentCoverage", func(t *testing.T) {
				e := New(t)
				sourcePath := test.path

				coverRegex := regexp.MustCompile(`coverage: (\d+.\d)% of statements`)

				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()

				cmd := exec.CommandContext(ctx, "go", "test", "-cover", sourcePath)
				cmd.Env = os.Environ() // pass parent envs
				cmd.Dir = sourcePath
				out, err := cmd.CombinedOutput()
				e.NoError(err, "Невозможно получить результат выполнения команды: %s. Вывод:\n\n %s", cmd, out)

				matched := coverRegex.Match(out)
				e.True(matched, "Отсутствует информация о покрытии кода тестами, команда: %s", cmd)
				e.Logf("Вывод команды:\n\n%s", string(out))
			})
		})
	}
}

type Iteration2BSuite struct {
	suite.Suite

	coverRegex *regexp.Regexp
}

func (suite *Iteration2BSuite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")

	regex, err := regexp.Compile(`coverage: (\d+.\d)% of statements`)
	suite.Require().NoError(err)

	suite.coverRegex = regex
}

// TestFilesPresence attempts to recursively find at least one Go test file in source path
func (suite *Iteration2BSuite) TestFilesPresence() {
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
		suite.T().Errorf("Не найден ни один тестовый файл по пути %s", flagTargetSourcePath)
		return
	}
	suite.T().Errorf("Неожиданная ошибка при поиске тестовых файлов по пути %s: %s", flagTargetSourcePath, err)
}

// TestServerCoverage attempts to obtain and parse coverage report using standard Go tooling
func (suite *Iteration2BSuite) TestServerCoverage() {
}
