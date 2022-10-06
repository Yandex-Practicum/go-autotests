package main

// Basic imports
import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/suite"
)

// Iteration15Suite является сьютом с тестами и состоянием для инкремента
type Iteration15Suite struct {
	suite.Suite
}

// SetupSuite подготавливает необходимые зависимости
func (suite *Iteration15Suite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")
	// pprof flags
	// suite.Require().NotEmpty(flagBaseProfilePath, "-base-profile-path non-empty flag required")
	// suite.Require().NotEmpty(flagResultProfilePath, "-result-profile-path non-empty flag required")
	// suite.Require().NotEmpty(flagPackageName, "-package-name non-empty flag required")
}

// TestBenchmarksPresence пробует запустить бенчмарки и получить результаты используя стандартный тулинг
func (suite *Iteration15Suite) TestBenchmarksPresence() {
	sourcePath := strings.TrimRight(flagTargetSourcePath, "/") + "/..."

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "test", "-bench=.", "-benchtime=100ms", "-run=^$", sourcePath)
	cmd.Env = os.Environ() // pass parent envs
	out, err := cmd.CombinedOutput()
	suite.Assert().NoError(err, "Невозможно получить результат выполнения команды: %s. Вывод:\n\n %s", cmd, out)

	matched := strings.Contains(string(out), "ns/op") && strings.Contains(string(out), "B/op")
	found := suite.Assert().True(matched, "Отсутствует информация о наличии бенчмарков в коде, команда: %s", cmd)

	if !found {
		suite.T().Logf("Вывод команды:\n\n%s", string(out))
	}
}

// TestProfilesDiff пробует получить разницу между двумя результатами запуска pprof
func (suite *Iteration15Suite) TestProfilesDiff() {
	suite.T().Skip("not implemented")

	baseFd, err := os.Open(flagBaseProfilePath)
	suite.Require().NoError(err, "Невозможно открыть файл с базовым профилем: %s", flagBaseProfilePath)
	defer baseFd.Close()

	resultFd, err := os.Open(flagResultProfilePath)
	suite.Require().NoError(err, "Невозможно открыть файл с результирующим профилем: %s", flagResultProfilePath)
	defer resultFd.Close()

	baseProfile, err := profile.Parse(baseFd)
	suite.Assert().NoError(err, "Невозможно распарсить базовый профиль")

	resultProfile, err := profile.Parse(resultFd)
	suite.Assert().NoError(err, "Невозможно распарсить результирующий профиль")

	baseProfile.Scale(-1)
	mergedProfile, err := profile.Merge([]*profile.Profile{resultProfile, baseProfile})

	// inspect only target package functions samples
	for i, sample := range mergedProfile.Sample {
		if len(mergedProfile.Function) < i {
			break
		}

		fn := mergedProfile.Function[i]
		fName := strings.ToLower(fn.Name)

		// inspect only target package non-test functions
		if !strings.Contains(fName, flagPackageName) ||
			strings.Contains(fName, "test_run") {
			continue
		}

		for _, value := range sample.Value {
			if value < 0 {
				return
			}
		}
	}

	suite.T().Error("Не удалось обнаружить положительных изменений в результирующем профиле")
}
