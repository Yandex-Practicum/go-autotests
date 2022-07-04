package main

// Basic imports
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

// Iteration2bSuite is a suite of autotests
type Iteration2bSuite struct {
	suite.Suite

	coverRegex *regexp.Regexp
}

// SetupSuite bootstraps suite dependencies
func (suite *Iteration2bSuite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")

	regex, err := regexp.Compile(`coverage: (\d+.\d)% of statements`)
	suite.Require().NoError(err)

	suite.coverRegex = regex
}

// TestFilesPresence attempts to recursively find at least one Go test file in source path
func (suite *Iteration2bSuite) TestFilesPresence() {
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
func (suite *Iteration2bSuite) TestServerCoverage() {
	sourcePath := strings.TrimRight(flagTargetSourcePath, "/") + "/..."

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "test", "-cover", sourcePath)
	cmd.Env = os.Environ() // pass parent envs
	out, err := cmd.CombinedOutput()
	suite.Assert().NoError(err, "Невозможно получить результат выполнения команды: %s. Вывод:\n\n %s", cmd, out)

	matched := suite.coverRegex.Match(out)
	found := suite.Assert().True(matched, "Отсутствует информация о покрытии кода тестами, команда: %s", cmd)

	if !found {
		suite.T().Logf("Вывод команды:\n\n%s", string(out))
	}
}
