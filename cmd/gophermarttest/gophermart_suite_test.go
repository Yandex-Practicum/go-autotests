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

// GophermartSuite is a suite of autotests
type GophermartSuite struct {
	suite.Suite

	gophermartServerAddress string
	gophermartProcess       *fork.BackgroundProcess

	accrualServerAddress string
	accrualProcess       *fork.BackgroundProcess
}

// SetupSuite bootstraps suite dependencies
func (suite *GophermartSuite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagGophermartBinaryPath, "-gophermart-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagGophermartDatabaseURI, "-gophermart-database-uri non-empty flag required")
	suite.Require().NotEmpty(flagGophermartHost, "-gophermart-host non-empty flag required")
	suite.Require().NotEmpty(flagGophermartPort, "-gophermart-port non-empty flag required")

	suite.Require().NotEmpty(flagAccrualBinaryPath, "-accrual-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagAccrualDatabaseURI, "-accrual-database-uri non-empty flag required")
	suite.Require().NotEmpty(flagAccrualHost, "-accrual-host non-empty flag required")
	suite.Require().NotEmpty(flagAccrualPort, "-accrual-port non-empty flag required")

	// start accrual server
	{
		suite.accrualServerAddress = "http://" + flagAccrualHost + ":" + flagAccrualPort

		envs := append(os.Environ(),
			"RUN_ADDRESS="+flagAccrualHost+":"+flagAccrualPort,
			"DATABASE_URI="+flagAccrualDatabaseURI,
		)
		p := fork.NewBackgroundProcess(context.Background(), flagAccrualBinaryPath,
			fork.WithEnv(envs...),
		)

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		err := p.Start(ctx)
		if err != nil {
			suite.T().Errorf("Невозможно запустить процесс командой %s: %s. Переменные окружения: %+v", p, err, envs)
			return
		}

		suite.accrualProcess = p

		port := flagAccrualPort
		err = p.WaitPort(ctx, "tcp", port)
		if err != nil {
			suite.T().Errorf("Не удалось дождаться пока порт %s станет доступен для запроса: %s", port, err)

			return
		}
	}

	// start gophermart server
	{
		suite.gophermartServerAddress = "http://" + flagGophermartHost + ":" + flagGophermartPort

		envs := append(os.Environ(),
			"RUN_ADDRESS="+flagGophermartHost+":"+flagGophermartPort,
			"DATABASE_URI="+flagGophermartDatabaseURI,
			"ACCRUAL_SYSTEM_ADDRESS="+suite.accrualServerAddress,
		)
		p := fork.NewBackgroundProcess(context.Background(), flagGophermartBinaryPath,
			fork.WithEnv(envs...),
		)

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		err := p.Start(ctx)
		if err != nil {
			suite.T().Errorf("Невозможно запустить процесс командой %s: %s. Переменные окружения: %+v", p, err, envs)
			return
		}

		suite.gophermartProcess = p

		port := flagGophermartPort
		err = p.WaitPort(ctx, "tcp", port)
		if err != nil {
			suite.T().Errorf("Не удалось дождаться пока порт %s станет доступен для запроса: %s", port, err)
			return
		}
	}
}

// TearDownSuite teardowns suite dependencies
func (suite *GophermartSuite) TearDownSuite() {
	suite.T().Logf("останавливаем процесс gophermart")
	suite.stopBinaryProcess(suite.gophermartProcess)

	suite.T().Log("останавливаем процесс accrual")
	suite.stopBinaryProcess(suite.accrualProcess)
}

func (suite *GophermartSuite) stopBinaryProcess(p *fork.BackgroundProcess) {
	if p == nil {
		return
	}

	exitCode, err := p.Stop(syscall.SIGINT, syscall.SIGKILL)
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

	out := p.Stderr(ctx)
	if len(out) > 0 {
		suite.T().Logf("Получен STDERR лог процесса:\n\n%s", string(out))
	}
	out = p.Stdout(ctx)
	if len(out) > 0 {
		suite.T().Logf("Получен STDOUT лог процесса:\n\n%s", string(out))
	}
}
