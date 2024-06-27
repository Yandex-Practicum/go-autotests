package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/suite"
)

type Iteration1Suite struct {
	suite.Suite

	serverAddress string
	serverProcess *fork.BackgroundProcess
}

func (suite *Iteration1Suite) SetupSuite() {
	// Проверяем необходимые флаги
	suite.Require().NotEmpty(flagServerBinaryPath, "-binary-path non-empty flag required")

	suite.serverAddress = "http://localhost:8080"

	// Для обеспечения обратной совместимости с будущими заданиями
	envs := append(os.Environ(), []string{
		"RESTORE=false",
	}...)
	suite.serverProcess = fork.NewBackgroundProcess(context.Background(), flagServerBinaryPath,
		fork.WithEnv(envs...),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	err := suite.serverProcess.Start(ctx)
	if err != nil {
		suite.T().Errorf("Невозможно запустить процесс командой %s: %s. Переменные окружения: %+v", suite.serverProcess, err, envs)
		return
	}

	port := "8080"
	err = suite.serverProcess.WaitPort(ctx, "tcp", port)
	if err != nil {
		suite.T().Errorf("Не удалось дождаться пока порт %s станет доступен для запроса: %s", port, err)
		return
	}
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

// TestHandlers имеет следующую схему работы для каждого типа метрики (gauge и counter):
// - формирует корректный запрос на обновление значения и ожидает http.StatusOK
// - формирует не корректный запрос без id-метрики и ожидает http.StatusNotFound
// - формирует не корректный запрос с value и ожидает http.StatusBadRequest

func (suite *Iteration1Suite) TestGaugeHandlers() {
	httpc := resty.New().SetHostURL(suite.serverAddress)

	suite.Run("update", func() {
		req := httpc.R().SetHeader("Content-Type", "text/plain")
		resp, err := req.Post("update/gauge/testGauge/100")

		noRespErr := suite.Assert().NoError(err,
			"Ошибка при попытке сделать запрос с обновлением gauge")

		validStatus := suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

		if !noRespErr || !validStatus {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("without id", func() {
		req := httpc.R().SetHeader("Content-Type", "text/plain")
		resp, err := req.Post("update/gauge/")

		noRespErr := suite.Assert().NoError(err,
			"Ошибка при попытке сделать запрос с обновлением gauge")

		validStatus := suite.Assert().Equalf(http.StatusNotFound, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

		if !noRespErr || !validStatus {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("invalid value", func() {
		req := httpc.R().SetHeader("Content-Type", "text/plain")
		resp, err := req.Post("update/gauge/testGauge/none")

		noRespErr := suite.Assert().NoError(err,
			"Ошибка при попытке сделать запрос с обновлением gauge")

		validStatus := suite.Assert().Equalf(http.StatusBadRequest, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

		if !noRespErr || !validStatus {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})
}

func (suite *Iteration1Suite) TestCounterHandlers() {
	httpc := resty.New().SetHostURL(suite.serverAddress)

	suite.Run("update", func() {
		req := httpc.R().SetHeader("Content-Type", "text/plain")
		resp, err := req.Post("update/counter/testCounter/100")

		noRespErr := suite.Assert().NoError(err,
			"Ошибка при попытке сделать запрос с обновлением counter")

		validStatus := suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

		if !noRespErr || !validStatus {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("without id", func() {
		req := httpc.R().SetHeader("Content-Type", "text/plain")
		resp, err := req.Post("update/counter/")

		noRespErr := suite.Assert().NoError(err,
			"Ошибка при попытке сделать запрос с обновлением counter")

		validStatus := suite.Assert().Equalf(http.StatusNotFound, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

		if !noRespErr || !validStatus {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("invalid value", func() {
		req := httpc.R().SetHeader("Content-Type", "text/plain")
		resp, err := req.Post("update/counter/testCounter/none")

		noRespErr := suite.Assert().NoError(err,
			"Ошибка при попытке сделать запрос с обновлением counter")

		validStatus := suite.Assert().Equalf(http.StatusBadRequest, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

		if !noRespErr || !validStatus {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})
}

func (suite *Iteration1Suite) TestUnknownHandlers() {
	httpc := resty.New().SetHostURL(suite.serverAddress)

	suite.Run("update invalid type", func() {
		req := httpc.R().SetHeader("Content-Type", "text/plain")
		resp, err := req.Post("update/unknown/testCounter/100")

		noRespErr := suite.Assert().NoError(err,
			"Ошибка при попытке сделать запрос с не корректным типом метрики")

		validStatus := suite.Assert().Containsf([]int{http.StatusBadRequest, http.StatusNotImplemented}, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

		if !noRespErr || !validStatus {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("update invalid method", func() {
		req := httpc.R().SetHeader("Content-Type", "text/plain")
		resp, err := req.Post("updater/counter/testCounter/100")

		noRespErr := suite.Assert().NoError(err,
			"Ошибка при попытке сделать запрос с не корректным типом метрики")

		validStatus := suite.Assert().Containsf([]int{http.StatusBadRequest, http.StatusNotFound}, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

		if !noRespErr || !validStatus {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})
}
