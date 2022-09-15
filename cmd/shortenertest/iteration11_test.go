package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/stdlib"
	"github.com/stretchr/testify/suite"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
)

// Iteration11Suite является сьютом с тестами и состоянием для инкремента
type Iteration11Suite struct {
	suite.Suite

	serverAddress string
	serverProcess *fork.BackgroundProcess

	dbconn *sql.DB
}

// SetupSuite подготавливает необходимые зависимости
func (suite *Iteration11Suite) SetupSuite() {
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

	// connect to database
	{
		// disable prepared statements
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
func (suite *Iteration11Suite) TearDownSuite() {
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

// TestInspectDatabase attempts to:
// - generate and send random URL to shorten handler
// - inspect database to find original URL record
func (suite *Iteration11Suite) TestInspectDatabase() {
	originalURL := generateTestURL(suite.T())

	httpc := resty.New().
		SetHostURL(suite.serverAddress)

	suite.Run("shorten", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req := httpc.R().
			SetContext(ctx).
			SetBody(originalURL)
		_, err := req.Post("/")
		noRespErr := suite.Assert().NoError(err, "Ошибка при попытке сделать запрос для сокращения URL")

		if !noRespErr {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("inspect", func() {
		suite.Require().NotNil(suite.dbconn, "Невозможно проинспектировать базу данных, нет подключения")

		tables, err := suite.fetchTables()
		suite.Require().NoError(err, "Ошибка получения списка таблиц базы данных")
		suite.Require().NotEmpty(tables, "Не найдено ни одной пользовательской таблицы в БД")

		var found bool
		for _, table := range tables {
			found, err = suite.findInTable(table, originalURL)
			if err != nil {
				suite.T().Logf("Ошибка поиска в таблице %s: %s", table, err)
			}
			if found {
				break
			}
		}

		suite.Require().Truef(found,
			"Не удалось обнаружить запись с оригинальным URL ни в одной таблице базы данных. Оригинальный URL: %s", originalURL)
	})
}

func (suite *Iteration11Suite) fetchTables() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	query := `
		SELECT
			table_schema || '.' || table_name
		FROM information_schema.tables
		WHERE
			table_schema NOT IN ('pg_catalog', 'information_schema')
	`

	rows, err := suite.dbconn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("не удалось выполнить запрос листинга таблиц: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tablename string
		if err := rows.Scan(&tablename); err != nil {
			return nil, fmt.Errorf("не удалось получить строку результата: %w", err)
		}
		tables = append(tables, tablename)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка обработки курсора базы данных: %w", err)
	}
	return tables, nil
}

func (suite *Iteration11Suite) findInTable(table, url string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	url = "%" + url + "%"

	query := `
		SELECT true
		FROM ` + table + ` AS tbl
		WHERE
			tbl::text LIKE $1
		LIMIT 1
	`

	var found bool
	err := suite.dbconn.QueryRowContext(ctx, query, url).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return found, err
}
