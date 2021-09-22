package main

// Basic imports
import (
	"context"
	"errors"
	"os"
	"syscall"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
)

// Iteration1Suite is a suite of autotests
type Iteration1Suite struct {
	suite.Suite

	serverAddress string
	serverProcess *fork.BackgroundProcess
}

// SetupSuite bootstraps suite dependencies
func (suite *Iteration1Suite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagTargetBinaryPath, "-binary-path non-empty flag required")

	suite.serverAddress = "http://localhost:8080"

	// start server
	{
		envs := append(os.Environ(), "DATABASE_DSN="+flagDatabaseDSN)
		p := fork.NewBackgroundProcess(context.Background(), flagTargetBinaryPath,
			fork.WithEnv(envs...),
		)

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		err := p.Start(ctx)
		if err != nil {
			suite.T().Errorf("Невозможно запустить процесс командой %s: %s. Переменные окружения: %+v", p, err, envs)
			return
		}

		port := "8080"
		err = p.ListenPort(ctx, "tcp", port)
		if err != nil {
			suite.T().Errorf("Не удалось дождаться пока на порт %s начнут поступать данные: %s", port, err)
			return
		}

		suite.serverProcess = p
	}
}

// TearDownSuite teardowns suite dependencies
func (suite *Iteration1Suite) TearDownSuite() {
	if suite.serverProcess == nil {
		return
	}

	exitCode, err := suite.serverProcess.Stop(syscall.SIGINT, syscall.SIGKILL)
	if err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			return
		}
		suite.T().Logf("Не удалось остановить процесс с помощью сигнала ОС: %s", err)
		return
	}

	if exitCode > 0 {
		suite.T().Logf("Процесс завершился с не нулевым статусом %d: %s", exitCode, err)
	}

	// try to read stdout/stderr
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	out := suite.serverProcess.Stderr(ctx)
	if len(out) > 0 {
		suite.T().Logf("Получен STDERR лог процесса:\n\n%s", string(out))
	}
	out = suite.serverProcess.Stdout(ctx)
	if len(out) > 0 {
		suite.T().Logf("Получен STDOUT лог процесса:\n\n%s", string(out))
	}
}

// TestAgent проверяет
// агент успешно стартует и передает какие-то данные по tcp, на 127.0.0.1:8080
func (suite *Iteration1Suite) TestAgent() {
	suite.Run("receive data from agent", func() {
	})
}
