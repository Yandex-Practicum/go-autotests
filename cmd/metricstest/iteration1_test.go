package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/suite"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
)

func TestIteration1(t *testing.T) {
	t.Run("Server", func(t *testing.T) {
		t.Run("TestCounterHandlers", func(t *testing.T) {
			t.Run("update", func(t *testing.T) {
				e := New(t)
				c := Client1(e)
				req := c.R()
				resp, err := req.Post("update/gauge/testGauge/100")
				e.NoError(err, "Ошибка при выполнении запроса")

			})
		})
	})
}

func Client1(e *Env) *resty.Client {
	StartProcessWhichListenPort(e, ServerHost(e), ServerPort(e), "metric server", ServerFilePath(e))
	return RestyClient(e, ServerAddress(e))
}

type Iteration1Suite struct {
	suite.Suite

	serverAddress string
	serverProcess *fork.BackgroundProcess
}

func (suite *Iteration1Suite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagServerBinaryPath, "-binary-path non-empty flag required")

	suite.serverAddress = "http://localhost:8080"

	envs := append(os.Environ(), []string{
		"RESTORE=false",
	}...)
	p := fork.NewBackgroundProcess(context.Background(), flagServerBinaryPath,
		fork.WithEnv(envs...),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	err := p.Start(ctx)
	if err != nil {
		suite.T().Errorf("Невозможно запустить процесс командой %s: %s. Переменные окружения: %+v", p, err, envs)
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

func (suite *Iteration1Suite) TearDownSuite() {
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

// TestHandlers проверяет
// сервер успешно стартует и открывет tcp порт 8080 на 127.0.0.1
// обработку POST запросов вида: ?id=<ID>&value=<VALUE>&type=<gauge|counter>
// а так же негативкейсы, запросы в которых отсутствуют id, value и задан не корректный type
func (suite *Iteration1Suite) TestGaugeHandlers() {
	// create HTTP client without redirects support
	errRedirectBlocked := errors.New("HTTP redirect blocked")
	redirPolicy := resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
		return errRedirectBlocked
	})

	httpc := resty.New().
		SetHostURL(suite.serverAddress).
		SetRedirectPolicy(redirPolicy)

	suite.Run("update", func() {
		req := httpc.R()
		resp, err := req.Post("update/gauge/testGauge/100")

		noRespErr := suite.Assert().NoError(err, "Ошибка при попытке сделать запрос с обновлением gauge")

		validStatus := suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

		if !noRespErr || !validStatus {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("without id", func() {
		req := httpc.R()
		resp, err := req.Post("update/gauge/")

		noRespErr := suite.Assert().NoError(err, "Ошибка при попытке сделать запрос с обновлением gauge")

		validStatus := suite.Assert().Equalf(http.StatusNotFound, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

		if !noRespErr || !validStatus {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("invalid value", func() {
		req := httpc.R()
		resp, err := req.Post("update/gauge/testGauge/none")

		noRespErr := suite.Assert().NoError(err, "Ошибка при попытке сделать запрос с обновлением gauge")

		validStatus := suite.Assert().Equalf(http.StatusBadRequest, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

		if !noRespErr || !validStatus {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})
}

func (suite *Iteration1Suite) TestCounterHandlers() {
	// create HTTP client without redirects support
	errRedirectBlocked := errors.New("HTTP redirect blocked")
	redirPolicy := resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
		return errRedirectBlocked
	})

	httpc := resty.New().
		SetHostURL(suite.serverAddress).
		SetRedirectPolicy(redirPolicy)

	suite.Run("update", func() {
		req := httpc.R()
		resp, err := req.Post("update/counter/testCounter/100")

		noRespErr := suite.Assert().NoError(err, "Ошибка при попытке сделать запрос с обновлением counter")

		validStatus := suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

		if !noRespErr || !validStatus {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("without id", func() {
		req := httpc.R()
		resp, err := req.Post("update/counter/")

		noRespErr := suite.Assert().NoError(err, "Ошибка при попытке сделать запрос с обновлением counter")

		validStatus := suite.Assert().Equalf(http.StatusNotFound, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

		if !noRespErr || !validStatus {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("invalid value", func() {
		req := httpc.R()
		resp, err := req.Post("update/counter/testCounter/none")

		noRespErr := suite.Assert().NoError(err, "Ошибка при попытке сделать запрос с обновлением counter")

		validStatus := suite.Assert().Equalf(http.StatusBadRequest, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

		if !noRespErr || !validStatus {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})
}

func (suite *Iteration1Suite) TestUnknownHandlers() {
	errRedirectBlocked := errors.New("HTTP redirect blocked")
	redirPolicy := resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
		return errRedirectBlocked
	})

	httpc := resty.New().
		SetHostURL(suite.serverAddress).
		SetRedirectPolicy(redirPolicy)

	suite.Run("update invalid type", func() {
		req := httpc.R()
		resp, err := req.Post("update/unknown/testCounter/100")

		noRespErr := suite.Assert().NoError(err, "Ошибка при попытке сделать запрос с не корректным типом метрики")

		validStatus := suite.Assert().Equalf(http.StatusNotImplemented, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

		if !noRespErr || !validStatus {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("update invalid method", func() {
		req := httpc.R()
		resp, err := req.Post("updater/counter/testCounter/100")

		noRespErr := suite.Assert().NoError(err, "Ошибка при попытке сделать запрос с не корректным типом метрики")

		validStatus := suite.Assert().Equalf(http.StatusNotFound, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

		if !noRespErr || !validStatus {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})
}
