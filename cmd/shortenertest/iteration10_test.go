package main

// Basic imports
import (
	"context"
	"errors"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/suite"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
)

// Iteration10Suite is a suite of autotests
type Iteration10Suite struct {
	suite.Suite

	serverAddress  string
	serverProcess  *fork.BackgroundProcess
	knownLibraries []string
}

// SetupSuite bootstraps suite dependencies
func (suite *Iteration10Suite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")
	suite.Require().NotEmpty(flagTargetBinaryPath, "-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagDatabaseDSN, "-database-dsn non-empty flag required")

	suite.serverAddress = "http://localhost:8080"
	suite.knownLibraries = []string{
		"database/sql",
		"github.com/jackc/pgx",
		"github.com/lib/pq",
	}

	// start server
	{
		envs := os.Environ()
		args := []string{"-d=" + flagDatabaseDSN}
		p := fork.NewBackgroundProcess(context.Background(), flagTargetBinaryPath,
			fork.WithEnv(envs...),
			fork.WithArgs(args...),
		)

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		err := p.Start(ctx)
		if err != nil {
			suite.T().Errorf("Невозможно запустить процесс командой %s: %s. Переменные окружения: %+v, аргументы: %+v", p, err, envs, args)
			return
		}

		port := "8080"
		err = p.WaitPort(ctx, "tcp", port)
		if err != nil {
			suite.T().Errorf("Не удалось дождаться пока порт %s станет доступен для запроса: %s", port, err)
			return
		}

		suite.serverProcess = p
	}
}

// TearDownSuite teardowns suite dependencies
func (suite *Iteration10Suite) TearDownSuite() {
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
		suite.T().Logf("Процесс завершился с не нулевым статусом: %s", err)

		// try to read stderr
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		out := suite.serverProcess.Stderr(ctx)
		if len(out) > 0 {
			suite.T().Logf("Получен лог процесса:\n\n%s", string(out))
		}

		return
	}
}

// TestLibraryUsage attempts to recursively find usage of database/sql in given sources
func (suite *Iteration10Suite) TestLibraryUsage() {
	err := usesKnownPackage(suite.T(), ".", suite.knownLibraries)
	if err == nil {
		return
	}
	if errors.Is(err, errUsageNotFound) {
		suite.T().Errorf("Не найдено использование библиотеки database/sql по пути %s", flagTargetSourcePath)
		return
	}
	suite.T().Errorf("Неожиданная ошибка при поиске использования библиотеки database/sql по пути %s: %s", flagTargetSourcePath, err)
}

// TestPingHandler attempts to call for ping handler and check positive result
func (suite *Iteration10Suite) TestPingHandler() {
	httpc := resty.New().
		SetHostURL(suite.serverAddress)

	ticker := time.NewTicker(time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			suite.T().Error("Не удалось получить код ответа 200 от хендлера 'GET /ping' за отведенное время")
			return
		case <-ticker.C:
			resp, _ := httpc.R().Get("/ping")
			if resp != nil && resp.StatusCode() == http.StatusOK {
				return
			}
		}
	}
}
