package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/suite"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
)

type Iteration5Suite struct {
	suite.Suite

	serverAddress string
	serverProcess *fork.BackgroundProcess
	agentProcess  *fork.BackgroundProcess
}

func (suite *Iteration5Suite) SetupSuite() {
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")
	suite.Require().NotEmpty(flagServerBinaryPath, "-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagAgentBinaryPath, "-agent-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagServerPort, "-server-port non-empty flag required")

	suite.serverAddress = "http://localhost:" + flagServerPort

	envs := append(os.Environ(), []string{
		"ADDRESS=localhost:" + flagServerPort,
		"REPORT_INTERVAL=5",
		"POLL_INTERVAL=1",

		// Для обеспечения обратной совместимости с будущими заданиями
		"RESTORE=false",
	}...)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	suite.agentUp(ctx, envs, flagServerPort)
	suite.serverUp(ctx, envs, flagServerPort)
}

func (suite *Iteration5Suite) serverUp(ctx context.Context, envs []string, port string) {
	suite.serverProcess = fork.NewBackgroundProcess(context.Background(), flagServerBinaryPath,
		fork.WithEnv(envs...),
	)

	err := suite.serverProcess.Start(ctx)
	if err != nil {
		suite.T().Errorf("Невозможно запустить процесс командой %q: %s. Переменные окружения: %+v", suite.serverProcess, err, envs)
		return
	}

	err = suite.serverProcess.WaitPort(ctx, "tcp", port)
	if err != nil {
		suite.T().Errorf("Не удалось дождаться пока порт %s станет доступен для запроса: %s", port, err)
		return
	}
}

func (suite *Iteration5Suite) agentUp(ctx context.Context, envs []string, port string) {
	suite.agentProcess = fork.NewBackgroundProcess(context.Background(), flagAgentBinaryPath,
		fork.WithEnv(envs...),
	)

	err := suite.agentProcess.Start(ctx)
	if err != nil {
		suite.T().Errorf("Невозможно запустить процесс командой %s: %s. Переменные окружения: %+v", suite.agentProcess, err, envs)
		return
	}

	err = suite.agentProcess.ListenPort(ctx, "tcp", port)
	if err != nil {
		suite.T().Errorf("Не удалось дождаться пока на порт %s начнут поступать данные: %s", port, err)
		return
	}
}

func (suite *Iteration5Suite) TearDownSuite() {
	suite.agentShutdown()
	suite.serverShutdown()
}

func (suite *Iteration5Suite) serverShutdown() {
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

func (suite *Iteration5Suite) agentShutdown() {
	if suite.agentProcess == nil {
		return
	}

	exitCode, err := suite.agentProcess.Stop(syscall.SIGINT, syscall.SIGKILL)
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

	out := suite.agentProcess.Stderr(ctx)
	if len(out) > 0 {
		suite.T().Logf("Получен STDERR лог процесса:\n\n%s", string(out))
	}
	out = suite.agentProcess.Stdout(ctx)
	if len(out) > 0 {
		suite.T().Logf("Получен STDOUT лог процесса:\n\n%s", string(out))
	}
}

func (suite *Iteration5Suite) TestGauge() {
	httpc := resty.NewWithClient(&http.Client{
		Transport: &http.Transport{
			DisableCompression: true,
		},
	}).SetHostURL(suite.serverAddress)

	count := 3
	suite.Run("update sequence", func() {
		id := strconv.Itoa(rand.Intn(256))
		req := httpc.R()
		for i := 0; i < count; i++ {
			v := strings.TrimRight(fmt.Sprintf("%.3f", rand.Float64()*1000000), "0")
			v = strings.TrimRight(v, ".")
			resp, err := req.Post("update/gauge/testSetGet" + id + "/" + v)
			noRespErr := suite.Assert().NoError(err,
				"Ошибка при попытке сделать запрос с обновлением gauge")

			validStatus := suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
				"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

			if !noRespErr || !validStatus {
				dump := dumpRequest(req.RawRequest, true)
				suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
			}

			resp, err = req.Get("value/gauge/testSetGet" + id)
			noRespErr = suite.Assert().NoError(err,
				"Ошибка при попытке сделать запрос для получения значения gauge")
			validStatus = suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
				"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)
			equality := suite.Assert().Equalf(v, resp.String(),
				"Несоответствие отправленного значения gauge (%s) полученному от сервера (%s), '%s %s'", v, resp.String(), req.Method, req.URL)

			if !noRespErr || !validStatus || !equality {
				dump := dumpRequest(req.RawRequest, true)
				suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
			}
		}
	})

	suite.Run("get unknown", func() {
		id := strconv.Itoa(rand.Intn(256))
		req := httpc.R()
		resp, err := req.Get("value/gauge/testUnknown" + id)
		noRespErr := suite.Assert().NoError(err,
			"Ошибка при попытке сделать запрос для получения значения gauge")
		validStatus := suite.Assert().Equalf(http.StatusNotFound, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

		if !noRespErr || !validStatus {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})
}

func (suite *Iteration5Suite) TestCounter() {
	httpc := resty.NewWithClient(&http.Client{
		Transport: &http.Transport{
			DisableCompression: true,
		},
	}).SetHostURL(suite.serverAddress)

	count := 3
	suite.Run("update sequence", func() {
		req := httpc.R()
		id := strconv.Itoa(rand.Intn(256))
		resp, err := req.Get("value/counter/testSetGet" + id)
		noRespErr := suite.Assert().NoError(err,
			"Ошибка при попытке сделать запрос для получения значения counter")

		if !noRespErr {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
			return
		}

		a, _ := strconv.ParseInt(resp.String(), 0, 64)

		for i := 0; i < count; i++ {
			v := rand.Intn(1024)
			a += int64(v)
			resp, err = req.Post("update/counter/testSetGet" + id + "/" + strconv.Itoa(v))

			noRespErr := suite.Assert().NoError(err,
				"Ошибка при попытке сделать запрос для обновления значения counter")
			validStatus := suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
				"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

			if !noRespErr || !validStatus {
				dump := dumpRequest(req.RawRequest, true)
				suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
				continue
			}

			resp, err := req.Get("value/counter/testSetGet" + id)
			noRespErr = suite.Assert().NoError(err,
				"Ошибка при попытке сделать запрос для получения значения counter")
			validStatus = suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
				"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)
			equality := suite.Assert().Equalf(fmt.Sprintf("%d", a), resp.String(),
				"Несоответствие отправленного значения counter (%d) полученному от сервера (%s), '%s %s'", a, resp.String(), req.Method, req.URL)

			if !noRespErr || !validStatus || !equality {
				dump := dumpRequest(req.RawRequest, true)
				suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
			}
		}
	})

	suite.Run("get unknown", func() {
		id := strconv.Itoa(rand.Intn(256))
		req := httpc.R()
		resp, err := req.Get("value/counter/testUnknown" + id)
		noRespErr := suite.Assert().NoError(err, "Ошибка при попытке сделать запрос для получения значения counter")
		validStatus := suite.Assert().Equalf(http.StatusNotFound, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

		if !noRespErr || !validStatus {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})
}
