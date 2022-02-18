package main

// Basic imports
import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/stretchr/testify/suite"
)

// Iteration15Suite is a suite of autotests
type Iteration15Suite struct {
	suite.Suite
}

// SetupSuite bootstraps suite dependencies
func (suite *Iteration15Suite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")
}

// TestBenchmarksPresence attempts to obtain and parse benchmarks report using standard Go tooling
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
