package main

// Basic imports
import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/stdlib"
	"github.com/stretchr/testify/suite"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
)

// Iteration11Suite is a suite of autotests
type Iteration11Suite struct {
	suite.Suite

	serverAddress  string
	serverPort     string
	serverProcess  *fork.BackgroundProcess
	serverArgs     []string
	knownLibraries []string

	rnd  *rand.Rand
	envs []string
	key  []byte

	dbconn *sql.DB
}

func (suite *Iteration11Suite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")
	suite.Require().NotEmpty(flagServerBinaryPath, "-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagAgentBinaryPath, "-agent-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagServerPort, "-server-port non-empty flag required")
	suite.Require().NotEmpty(flagDatabaseDSN, "-database-dsn non-empty flag required")
	suite.Require().NotEmpty(flagSHA256Key, "-key non-empty flag required")

	suite.rnd = rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
	suite.serverAddress = "http://localhost:" + flagServerPort
	suite.serverPort = flagServerPort

	suite.key = []byte(flagSHA256Key)

	suite.envs = append(os.Environ(), []string{
		"RESTORE=true",
		"DATABASE_DSN=" + flagDatabaseDSN,
	}...)

	suite.serverArgs = []string{
		"-a=localhost:" + flagServerPort,
		// "-s=5s",
		"-r=false",
		"-i=5m",
		"-k=" + flagSHA256Key,
		"-d=" + flagDatabaseDSN,
	}

	suite.knownLibraries = []string{
		"database/sql",
		"github.com/jackc/pgx",
		"github.com/lib/pq",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

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

	suite.serverUp(ctx, suite.envs, suite.serverArgs, flagServerPort)
}

func (suite *Iteration11Suite) serverUp(ctx context.Context, envs, args []string, port string) {
	p := fork.NewBackgroundProcess(context.Background(), flagServerBinaryPath,
		fork.WithEnv(envs...),
		fork.WithArgs(args...),
	)

	err := p.Start(ctx)
	if err != nil {
		suite.T().Errorf("Невозможно запустить процесс командой %q: %s. Переменные окружения: %+v, флаги командной строки: %+v", p, err, envs, args)
		return
	}

	err = p.WaitPort(ctx, "tcp", port)
	if err != nil {
		suite.T().Errorf("Не удалось дождаться пока порт %s станет доступен для запроса: %s", port, err)
		return
	}
	suite.serverProcess = p
}

// TearDownSuite teardowns suite dependencies
func (suite *Iteration11Suite) TearDownSuite() {
	suite.serverShutdown()
}

func (suite *Iteration11Suite) serverShutdown() {
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

// TestInspectDatabase attempts to:
// - generate and send random counter
// - inspect database to find original counter record
func (suite *Iteration11Suite) TestInspectDatabase() {
	id := "PopulateCounter" + strconv.Itoa(suite.rnd.Intn(256*256*256))

	httpc := resty.New().
		SetHostURL(suite.serverAddress)

	suite.Run("populate counter", func() {
		req := httpc.R().
			SetHeader("Content-Type", "application/json")

		var value int64
		resp, err := suite.SetHBody(req,
			&Metrics{
				ID:    id,
				MType: "counter",
				Delta: &value,
			}).
			Post("update/")

		dumpErr := suite.Assert().NoError(err, "Ошибка при попытке сделать запрос с обновлением counter")
		dumpErr = dumpErr && suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере %q: %q ", req.Method, req.URL)
		dumpErr = dumpErr && suite.Assert().NoError(err, "Ошибка при попытке сделать запрос для сокращения URL")

		if !dumpErr {
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
			found, err = suite.findInTable(table, id)
			if err != nil {
				suite.T().Logf("Ошибка поиска в таблице %s: %s", table, err)
			}
			if found {
				break
			}
		}

		suite.Require().Truef(found,
			"Не удалось обнаружить запись с оригинальной метрикой счетчика ни в одной таблице базы данных. Оригинальный ID метрики счетчика: %s", id)
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

func (suite *Iteration11Suite) findInTable(table, name string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	name = "%" + name + "%"

	query := `
		SELECT true
		FROM ` + table + ` AS tbl
		WHERE
			tbl::text LIKE $1
		LIMIT 1
	`

	var found bool
	err := suite.dbconn.QueryRowContext(ctx, query, name).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return found, err
}

func (suite *Iteration11Suite) SetHBody(r *resty.Request, m *Metrics) *resty.Request {
	hash := suite.Hash(m)
	m.Hash = hash
	return r.SetBody(m)
}

func (suite *Iteration11Suite) Hash(m *Metrics) string {
	var data string
	switch m.MType {
	case "counter":
		data = fmt.Sprintf("%s:%s:%d", m.ID, m.MType, *m.Delta)
	case "gauge":
		data = fmt.Sprintf("%s:%s:%f", m.ID, m.MType, *m.Value)
	}
	h := hmac.New(sha256.New, suite.key)
	h.Write([]byte(data))
	return fmt.Sprintf("%x", h.Sum(nil))
}
