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

// Iteration12Suite is a suite of autotests
type Iteration12Suite struct {
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

func (suite *Iteration12Suite) SetupSuite() {
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

// TearDownSuite teardowns suite dependencies
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

func (suite *Iteration12Suite) TestBatchAPI() {
	// create HTTP client without redirects support
	errRedirectBlocked := errors.New("HTTP redirect blocked")
	redirPolicy := resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
		return errRedirectBlocked
	})
	httpc := resty.New().
		SetHostURL(suite.serverAddress).
		SetRedirectPolicy(redirPolicy)

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
				MType: "counter"}).
			SetResult(&result).
			Post("value/")

		dumpErr := suite.Assert().NoError(err, "Ошибка при попытке сделать запрос с получением значения counter")
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
			Metrics{
				ID:    idCounter,
				MType: "counter",
				Delta: &valueCounter1,
			},
			Metrics{
				ID:    idGauge,
				MType: "gauge",
				Value: &valueGauge1,
			},
			Metrics{
				ID:    idCounter,
				MType: "counter",
				Delta: &valueCounter2,
			},
			Metrics{
				ID:    idGauge,
				MType: "gauge",
				Value: &valueGauge2,
			},
		}

		resp, err := suite.SetHBody(req, metrics).
			Post("updates/")

		dumpErr := suite.Assert().NoError(err, "Ошибка при попытке сделать запрос с обновлением списка метрик")
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
				MType: "counter"}).
			SetResult(&result).
			Post("value/")

		dumpErr := suite.Assert().NoError(err, "Ошибка при попытке сделать запрос с получением значения counter")
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
		dumpErr = dumpErr && suite.Equal(suite.Hash(&result), result.Hash, "Хеш-сумма не соответствует расчетной")

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

		dumpErr := suite.Assert().NoError(err, "Ошибка при попытке сделать запрос с получением значения gauge")
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
		dumpErr = dumpErr && suite.Equal(suite.Hash(&result), result.Hash, "Хеш-сумма не соответствует расчетной")

		if !dumpErr {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
			dump = dumpResponse(resp.RawResponse, true)
			suite.T().Logf("Оригинальный ответ:\n\n%s", dump)
		}
	})

}

func (suite *Iteration12Suite) SetHBody(r *resty.Request, l []Metrics) *resty.Request {
	for i, m := range l {
		hash := suite.Hash(&m)
		l[i].Hash = hash
	}
	return r.SetBody(&l)
}

func (suite *Iteration12Suite) Hash(m *Metrics) string {
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
