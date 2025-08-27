package main

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"syscall"
	"time"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/stdlib"
	"github.com/stretchr/testify/suite"
)

// Iteration16Suite является сьютом с тестами и состоянием для инкремента
type Iteration16Suite struct {
	suite.Suite

	serverAddress string
	serverProcess *fork.BackgroundProcess

	dbconn *sql.DB
}

// SetupSuite подготавливает необходимые зависимости
func (suite *Iteration16Suite) SetupSuite() {
	// проверяем наличие необходимых флагов
	suite.Require().NotEmpty(flagTargetBinaryPath, "-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagDatabaseDSN, "-database-dsn non-empty flag required")

	suite.serverAddress = "http://localhost:8080"

	// запускаем процесс тестируемого сервера
	{
		envs := os.Environ()
		args := []string{"-d=" + flagDatabaseDSN}

		p := fork.NewBackgroundProcess(context.Background(), flagTargetBinaryPath,
			fork.WithEnv(envs...),
			fork.WithArgs(args...),
		)
		suite.serverProcess = p

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
	}

	if flagDatabaseDSN != "" {
		driverConfig := stdlib.DriverConfig{
			ConnConfig: pgx.ConnConfig{
				PreferSimpleProtocol: true,
			},
		}
		stdlib.RegisterDriverConfig(&driverConfig)

		conn, err := sql.Open("pgx", driverConfig.ConnectionString(flagDatabaseDSN))
		if err != nil {
			suite.T().Errorf("Не удалось подключиться к базе данных: %s", err)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err = conn.PingContext(ctx); err != nil {
			suite.T().Errorf("Не удалось подключиться проверить подключение к базе данных: %s", err)
			return
		}

		suite.dbconn = conn
	}
}

// TearDownSuite высвобождает имеющиеся зависимости
func (suite *Iteration16Suite) TearDownSuite() {
	if suite.dbconn != nil {
		_ = suite.dbconn.Close()
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

	// получаем стандартные выводы (логи) процесса
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

func (suite *Iteration18Suite) TestTODO() {
	// no-op for now
}
