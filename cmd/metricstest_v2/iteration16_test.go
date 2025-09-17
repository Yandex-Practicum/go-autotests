package main

import (
	"context"
	"errors"
	"math/rand"
	"os"
	"syscall"
	"time"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
	"github.com/stretchr/testify/suite"
)

type Iteration16Suite struct {
	suite.Suite

	serverAddress string
	serverPort string
	serverProcess *fork.BackgroundProcess
	agentProcess *fork.BackgroundProcess

	rnd *rand.Rand
}

func (suite *Iteration16Suite) SetupSuite() {
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")
	suite.Require().NotEmpty(flagServerBinaryPath, "-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagAgentBinaryPath, "-agent-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagServerPort, "-server-port non-empty flag required")
	suite.Require().NotEmpty(flagDatabaseDSN, "-database-dsn non-empty flag required")

	suite.rnd = rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
	suite.serverAddress = "http://localhost:" + flagServerPort

	envs := append(os.Environ(), []string{
		"ADDRESS=localhost:" + flagServerPort,
		"RESTORE=true",
		"DATABASE_DSN=" + flagDatabaseDSN,
	}...)

	serverArgs := []string{}

	agentArgs := []string{}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	suite.agentUp(ctx, envs, agentArgs, flagServerPort)
	suite.serverUp(ctx, envs, serverArgs, flagServerPort)
}

func (suite *Iteration16Suite) serverUp(ctx context.Context, envs, args []string, port string) {
	suite.serverProcess = fork.NewBackgroundProcess(context.Background(), flagServerBinaryPath,
		fork.WithEnv(envs...), 
		fork.WithArgs(args...),
	)

	err := suite.serverProcess.Start(ctx)
	if err != nil {
		suite.T().Errorf("Невозможно запустить процесс командой %q: %s. Переменные окружения: %+v, флаги командной строки: %+v", suite.serverProcess, err, envs, args)
		return
	}

	err = suite.serverProcess.WaitPort(ctx, "tcp", port)
	if err != nil {
		suite.T().Errorf("Не удалось дождвться пока порт %s станет доступен для запроса: %s", port, err)
		return
	}
}

func (suite *Iteration16Suite) agentUp(ctx context.Context, envs, args [] string, port string) {
	suite.agentProcess = fork.NewBackgroundProcess(context.Background(), flagAgentBinaryPath,
		fork.WithEnv(envs...),
		fork.WithArgs(args...),
	)

	err := suite.agentProcess.Start(ctx)
	if err != nil {
		suite.T().Errorf("Невозможно запустить процесс командой %q: %s. Переменные окружения: %+v, флаги командной строки: %+v", suite.agentProcess, err, envs, args)
		return
	}

	err = suite.agentProcess.ListenPort(ctx, "tcp", port)
	if err != nil {
		suite.T().Errorf("Не удалось дождаться пока на порт %s начнут поступать данные: %s", port, err)
		return
	}
}

func (suite *Iteration16Suite) TearDownSuite() {
	suite.agentShutdown()
	suite.serverShutdown()
}

func (suite *Iteration16Suite) serverShutdown() {
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

func (suite *Iteration16Suite) agentShutdown() {
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

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	out := suite.agentProcess.Stderr(ctx)
	if len(out) > 0 {
		suite.T().Logf("Получен STDERR лог процесса:\n\n%s", string(out))
	}

	out = suite.agentProcess.Stdout(ctx)
	if len(out) > 0 {
		suite.T().Logf("получен STDOUT лог процесса:\n\n%s", string(out))
	}

}

func (suite *Iteration16Suite) TestTODO() {

}
