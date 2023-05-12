package main

// Basic imports
import (
	"context"
	"database/sql"
	"errors"
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

type Iteration12Suite struct {
	suite.Suite

	serverAddress string
	serverPort    string
	serverProcess *fork.BackgroundProcess

	rnd *rand.Rand

	dbconn *sql.DB
}

func (suite *Iteration12Suite) SetupSuite() {
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")
	suite.Require().NotEmpty(flagServerBinaryPath, "-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagAgentBinaryPath, "-agent-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagServerPort, "-server-port non-empty flag required")
	suite.Require().NotEmpty(flagDatabaseDSN, "-database-dsn non-empty flag required")

	suite.rnd = rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
	suite.serverAddress = "http://localhost:" + flagServerPort
	suite.serverPort = flagServerPort

	envs := append(os.Environ(), []string{
		"ADDRESS=localhost:" + flagServerPort,
		"RESTORE=true",
		"DATABASE_DSN=" + flagDatabaseDSN,
		"STORE_INTERVAL=1",
	}...)

	serverArgs := []string{
		"-r=false",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	{
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

	suite.serverUp(ctx, envs, serverArgs, flagServerPort)
}

func (suite *Iteration12Suite) serverUp(ctx context.Context, envs, args []string, port string) {
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

func (suite *Iteration12Suite) TearDownSuite() {
	suite.serverShutdown()
}

func (suite *Iteration12Suite) serverShutdown() {
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

func (suite *Iteration12Suite) TestBatchAPI() {
	httpc := resty.New().SetHostURL(suite.serverAddress)

	idCounter := "CounterBatchZip" + strconv.Itoa(suite.rnd.Intn(256))
	idGauge := "GaugeBatchZip" + strconv.Itoa(suite.rnd.Intn(256))
	valueCounter1, valueCounter2 := int64(suite.rnd.Int31()), int64(suite.rnd.Int31())
	var valueCounter0 int64
	valueGauge1, valueGauge2 := suite.rnd.Float64()*1e6, suite.rnd.Float64()*1e6

	req := httpc.R().
		SetHeader("Accept-Encoding", "gzip").
		SetHeader("Content-Type", "application/json")

	suite.Run("get random counter", func() {
		var result Metrics
		resp, err := req.
			SetBody(&Metrics{
				ID:    idCounter,
				MType: "counter",
			}).
			SetResult(&result).
			Post("value/")

		dumpErr := suite.Assert().NoError(err,
			"Ошибка при попытке сделать запрос с получением значения counter")
		switch resp.StatusCode() {
		case http.StatusOK:
			dumpErr = dumpErr && suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
				"Несоответствие статус кода ответа ожидаемому в хендлере %q: %q ", req.Method, req.URL)
			dumpErr = dumpErr && suite.Assert().Containsf(resp.Header().Get("Content-Type"), "application/json",
				"Заголовок ответа Content-Type содержит несоответствующее значение")
			dumpErr = dumpErr && suite.NotNil(result.Delta,
				"Получено не инициализированное значение Delta '%q %s'", req.Method, req.URL)
			valueCounter0 = *result.Delta
		case http.StatusNotFound:
		default:
			dumpErr = false
			suite.T().Fatalf("Несоответствие статус кода %d ответа ожидаемому http.StatusNotFound или http.StatusOK в хендлере %q: %q", resp.StatusCode(), req.Method, req.URL)
			return
		}
		if !dumpErr {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
			dump = dumpResponse(resp.RawResponse, true)
			suite.T().Logf("Оригинальный ответ:\n\n%s", dump)
		}
	})

	suite.Run("batch update random metrics", func() {
		metrics := []Metrics{
			{
				ID:    idCounter,
				MType: "counter",
				Delta: &valueCounter1,
			},
			{
				ID:    idGauge,
				MType: "gauge",
				Value: &valueGauge1,
			},
			{
				ID:    idCounter,
				MType: "counter",
				Delta: &valueCounter2,
			},
			{
				ID:    idGauge,
				MType: "gauge",
				Value: &valueGauge2,
			},
		}

		resp, err := req.SetBody(metrics).
			Post("updates/")

		dumpErr := suite.Assert().NoError(err,
			"Ошибка при попытке сделать запрос с обновлением списка метрик")
		dumpErr = dumpErr && suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере %q: %q ", req.Method, req.URL)

		if !dumpErr {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
			dump = dumpResponse(resp.RawResponse, true)
			suite.T().Logf("Оригинальный ответ:\n\n%s", dump)
		}
	})

	suite.Run("check counter value", func() {
		var result Metrics
		resp, err := req.
			SetBody(&Metrics{
				ID:    idCounter,
				MType: "counter",
			}).
			SetResult(&result).
			Post("value/")

		dumpErr := suite.Assert().NoError(err,
			"Ошибка при попытке сделать запрос с получением значения counter")
		dumpErr = dumpErr && suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере %q: %q ", req.Method, req.URL)
		dumpErr = dumpErr && suite.Assert().Containsf(resp.Header().Get("Content-Type"), "application/json",
			"Заголовок ответа Content-Type содержит несоответствующее значение")
		dumpErr = dumpErr && suite.Assert().Containsf(resp.Header().Get("Content-Encoding"), "gzip",
			"Заголовок ответа Content-Encoding содержит несоответствующее значение")
		dumpErr = dumpErr && suite.NotNil(result.Delta,
			"Несоответствие отправленного значения counter (r:%d+w:%d+w:%d) полученному от сервера (nil), '%q %s'", valueCounter0, valueCounter1, valueCounter2, req.Method, req.URL)
		dumpErr = dumpErr && suite.Assert().Equalf(valueCounter0+valueCounter1+valueCounter2, *result.Delta,
			"Несоответствие отправленного значения counter (r:%d+w:%d+w:%d) полученному от сервера (%d), '%q %s'", valueCounter0, valueCounter1, valueCounter2, *result.Delta, req.Method, req.URL)

		if !dumpErr {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
			dump = dumpResponse(resp.RawResponse, true)
			suite.T().Logf("Оригинальный ответ:\n\n%s", dump)
		}
	})

	suite.Run("check gauge value", func() {
		var result Metrics
		resp, err := req.
			SetBody(&Metrics{
				ID:    idGauge,
				MType: "gauge",
			}).
			SetResult(&result).
			Post("value/")

		dumpErr := suite.Assert().NoError(err,
			"Ошибка при попытке сделать запрос с получением значения gauge")
		dumpErr = dumpErr && suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере %q: %q ", req.Method, req.URL)
		dumpErr = dumpErr && suite.Assert().Containsf(resp.Header().Get("Content-Type"), "application/json",
			"Заголовок ответа Content-Type содержит несоответствующее значение")
		dumpErr = dumpErr && suite.Assert().Containsf(resp.Header().Get("Content-Encoding"), "gzip",
			"Заголовок ответа Content-Encoding содержит несоответствующее значение")
		dumpErr = dumpErr && suite.Assert().NotEqualf(nil, result.Value,
			"Несоответствие отправленного значения gauge (%f) полученному от сервера (nil), '%q %s'", valueGauge2, req.Method, req.URL)
		dumpErr = dumpErr && suite.Assert().Equalf(valueGauge2, *result.Value,
			"Несоответствие отправленного значения gauge (%f) полученному от сервера (%f), '%q %s'", valueGauge2, *result.Value, req.Method, req.URL)

		if !dumpErr {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
			dump = dumpResponse(resp.RawResponse, true)
			suite.T().Logf("Оригинальный ответ:\n\n%s", dump)
		}
	})
}
