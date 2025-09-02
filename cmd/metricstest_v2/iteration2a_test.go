package main

import (
	"context"
	"errors"
	"os"
	"syscall"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
)

type Iteration2ASuite struct {
	suite.Suite

	agentAddress string
	agentProcess *fork.BackgroundProcess
}

func (suite *Iteration2ASuite) SetupSuite() {
	suite.Require().NotEmpty(flagAgentBinaryPath, "-agent-binary-path non-empty flag required")

	suite.agentAddress = "http://localhost:8080"

	// Для обеспечения обратной совместимости с будущими заданиями
	envs := append(os.Environ(), []string{
		"RESTORE=false",
	}...)
	suite.agentProcess = fork.NewBackgroundProcess(context.Background(), flagAgentBinaryPath,
		fork.WithEnv(envs...),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	err := suite.agentProcess.Start(ctx)
	if err != nil {
		suite.T().Errorf("Невозможно запустить процесс командой %s: %s. Переменные окружения: %+v", suite.agentProcess, err, envs)
		return
	}

	port := "8080"
	err = suite.agentProcess.ListenPort(ctx, "tcp", port)
	if err != nil {
		suite.T().Errorf("Не удалось дождаться пока на порт %s начнут поступать данные: %s", port, err)
		return
	}
}

func (suite *Iteration2ASuite) TearDownSuite() {
	if suite.agentProcess == nil {
		return
	}

	exitCode, err := suite.agentProcess.Stop(syscall.SIGINT, syscall.SIGKILL)
	if err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			return
		}
		suite.T().Logf("Не удалось остановить процесс с помощью сигнала ОС: %s", err)
		return
	}

	if exitCode > 0 {
		suite.T().Logf("Процесс завершился с не нулевым статусом %d", exitCode)
	}

	// try to read stdout/stderr
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	out := suite.agentProcess.Stderr(ctx)
	if len(out) > 0 {
		suite.T().Logf("Получен STDERR лог процесса:\n\n%s", string(out))
	}
	out = suite.agentProcess.Stdout(ctx)
	if len(out) > 0 {
		suite.T().Logf("Получен STDOUT лог процесса:\n\n%s", string(out))
	}
}

// TestAgent проверяет:
// - агент успешно стартует и передает какие-нибудь данные по tcp, на localhost:8080
func (suite *Iteration2ASuite) TestAgent() {
	suite.Run("receive data from agent", func() {
	})
}
