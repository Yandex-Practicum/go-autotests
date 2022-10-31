package main

import (
	"bytes"
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
		err := suite.startServer()
		if err != nil {
			suite.T().Errorf("Не удалось запустить процесс сервера: %w", err)
			return
		}
	}

	// получаем соединение к БД
	{
		// отключаем prepared statements
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

		// будем ожидать ответа на пинг БД в течении 2 секунд
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// пингуем соединение с БД, чтобы убежиться, что оно живое
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

	out, err := suite.stopServer()
	if err != nil {
		suite.T().Logf("Процесс завершился с ошибкой: %w", err)
	}
	if len(out) > 0 {
		suite.T().Log(string(out))
	}
}

// TestInspectDatabase пробует:
// - сгенерировать псевдослучайный URL и послать его на сокращение
// - проинспектировать БД на наличие исходного URL
func (suite *Iteration11Suite) TestInspectDatabase() {
	// генерируем URL
	originalURL := generateTestURL(suite.T())

	httpc := resty.New().
		SetBaseURL(suite.serverAddress)

	// сокращаем URL
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

	// инспектируем БД
	suite.Run("inspect", func() {
		suite.inspectTables(originalURL)
	})

	// инспектируем БД после рестарта приложения
	suite.Run("check_after_restart", func() {
		out, err := suite.stopServer()
		noRestartErr := suite.Assert().NoError(err, "Не удалось остановить процесс сервера")

		if !noRestartErr && len(out) > 0 {
			suite.T().Log(string(out))
		}

		err = suite.startServer()
		suite.Require().NoError(err, "Не удалось перезапустить процесс сервера")

		suite.inspectTables(originalURL)
	})
}

func (suite *Iteration11Suite) startServer() error {
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
		return fmt.Errorf("Невозможно запустить процесс командой %s: %s. Переменные окружения: %+v, аргументы: %+v", p, err, envs, args)
	}

	// ожидаем пока порт будет занят
	port := "8080"
	err = p.WaitPort(ctx, "tcp", port)
	if err != nil {
		return fmt.Errorf("Не удалось дождаться пока порт %s станет доступен для запроса: %s", port, err)
	}

	suite.serverProcess = p
	return nil
}

// stopServer останавливает процесс сервера
func (suite *Iteration11Suite) stopServer() (log []byte, err error) {
	exitCode, err := suite.serverProcess.Stop(syscall.SIGINT, syscall.SIGKILL)
	if err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			return nil, nil
		}
		return nil, fmt.Errorf("Не удалось остановить процесс с помощью сигнала ОС: %w", err)
	}

	if exitCode > 0 {
		return nil, fmt.Errorf("Процесс завершился с не нулевым статусом %d", exitCode)
	}

	// получаем стандартные выводы (логи) процесса
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	var buf bytes.Buffer

	out := suite.serverProcess.Stderr(ctx)
	if len(out) > 0 {
		buf.WriteString("Получен STDERR лог процесса:\n\n")
		buf.Write(out)
	}
	out = suite.serverProcess.Stdout(ctx)
	if len(out) > 0 {
		buf.WriteString("Получен STDOUT лог процесса:\n\n")
		buf.Write(out)
	}

	return buf.Bytes(), nil
}

func (suite *Iteration11Suite) inspectTables(originalURL string) {
	suite.T().Helper()

	suite.Require().NotNil(suite.dbconn, "Невозможно проинспектировать базу данных, нет подключения")

	// получаем существющие таблицы в БД
	tables, err := suite.fetchTables()
	suite.Require().NoError(err, "Ошибка получения списка таблиц базы данных")
	suite.Require().NotEmpty(tables, "Не найдено ни одной пользовательской таблицы в БД")

	// инспектируем каждую таблицу по очереди
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
}

// fetchTables возвращает имеющиеся в БД таблицы
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

// findInTable ищет в заданной таблице необходимый URL
func (suite *Iteration11Suite) findInTable(table, url string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// искать будем по вхождению подстроки в строку
	url = "%" + url + "%"

	// `tbl::text` превращает всю запись в таблице в текстовую строку
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
		// не найдено ни одной записи в БД
		return false, nil
	}
	return found, err
}
