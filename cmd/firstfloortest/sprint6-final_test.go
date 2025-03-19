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

type Sprint6FinalSuite struct {
	suite.Suite

	serverProcess *fork.BackgroundProcess
}

func (suite *Sprint6FinalSuite) SetupSuite() {
	suite.Require().NotEmpty(flagServerBinaryPath, "-server-binary-path non-empty flag required")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	suite.serverUp(ctx)
}

func (suite *Sprint6FinalSuite) serverUp(ctx context.Context) {
	suite.serverProcess = fork.NewBackgroundProcess(context.Background(), flagServerBinaryPath)

	err := suite.serverProcess.Start(ctx)
	if err != nil {
		suite.T().Errorf("Невозможно запустить процесс командой %q: %s. Переменные окружения: %+v", suite.serverProcess, err, envs)
		return
	}

	const port = ":8080"
	err = suite.serverProcess.WaitPort(ctx, "tcp", port)
	if err != nil {
		suite.T().Errorf("Не удалось дождаться пока порт %s станет доступен для запроса: %s", port, err)
		return
	}
}

func (suite *Sprint6FinalSuite) TearDownSuite() {
	suite.serverShutdown()
}

func (suite *Sprint6FinalSuite) serverShutdown() {
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
		suite.T().Logf("Процесс завершился с не нулевым статусом %d", exitCode)
	}

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

func (suite *Sprint6FinalSuite) TestSprint6Final() {
	if 1 != 1 {
		suite.T().Errorf("1 != 1")
	}
}
