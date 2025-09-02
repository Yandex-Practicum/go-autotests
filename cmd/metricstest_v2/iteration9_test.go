package main

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/suite"
)

type Iteration9Suite struct {
	suite.Suite

	serverAddress string
	serverPort    string
	serverProcess *fork.BackgroundProcess
	agentProcess  *fork.BackgroundProcess

	rnd  *rand.Rand
	envs []string
}

func (suite *Iteration9Suite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")
	suite.Require().NotEmpty(flagServerBinaryPath, "-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagAgentBinaryPath, "-agent-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagServerPort, "-server-port non-empty flag required")
	suite.Require().NotEmpty(flagFileStoragePath, "-file-storage-path non-empty flag required")

	suite.rnd = rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
	suite.serverAddress = "http://localhost:" + flagServerPort
	suite.serverPort = flagServerPort

	suite.envs = append(os.Environ(), []string{
		"ADDRESS=localhost:" + flagServerPort,

		"RESTORE=true",
		"STORE_INTERVAL=2",
		"FILE_STORAGE_PATH=" + flagFileStoragePath,
	}...)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	suite.agentUp(ctx, suite.envs, flagServerPort)
	suite.serverUp(ctx, suite.envs, flagServerPort)
}

func (suite *Iteration9Suite) serverUp(ctx context.Context, envs []string, port string) {
	p := fork.NewBackgroundProcess(context.Background(), flagServerBinaryPath,
		fork.WithEnv(envs...),
	)

	err := p.Start(ctx)
	if err != nil {
		suite.T().Errorf("Невозможно запустить процесс командой %q: %s. Переменные окружения: %+v", p, err, envs)
		return
	}

	err = p.WaitPort(ctx, "tcp", port)
	if err != nil {
		suite.T().Errorf("Не удалось дождаться пока порт %s станет доступен для запроса: %s", port, err)
		return
	}
	suite.serverProcess = p
}

func (suite *Iteration9Suite) agentUp(ctx context.Context, envs []string, port string) {
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

func (suite *Iteration9Suite) TearDownSuite() {
	suite.agentShutdown()
	suite.serverShutdown()
}

func (suite *Iteration9Suite) serverShutdown() {
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

func (suite *Iteration9Suite) agentShutdown() {
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

func (suite *Iteration9Suite) TestCounterHandlers() {
	httpc := resty.New().SetHostURL(suite.serverAddress)

	id := "GetSet" + strconv.Itoa(suite.rnd.Intn(256))
	var storage int64

	suite.Run("update", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		value1, value2 := int64(suite.rnd.Int31()), int64(suite.rnd.Int31())
		req := httpc.R().
			SetHeader("Content-Type", "application/json")

		// Вдруг на сервере уже есть значение, на всякий случай запросим.
		var result Metrics
		resp, err := req.
			SetContext(ctx).
			SetBody(&Metrics{
				ID:    id,
				MType: "counter",
			}).
			SetResult(&result).
			Post("value/")

		dumpErr := suite.Assert().NoError(err,
			"Ошибка при попытке сделать запрос с получением значения counter")
		var value0 int64
		switch resp.StatusCode() {
		case http.StatusOK:
			dumpErr = dumpErr && suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
				"Несоответствие статус кода ответа ожидаемому в хендлере %q: %q ", req.Method, req.URL)
			dumpErr = dumpErr && suite.Assert().Containsf(resp.Header().Get("Content-Type"), "application/json",
				"Заголовок ответа Content-Type содержит несоответствующее значение")
			dumpErr = dumpErr && suite.NotNil(result.Delta,
				"Получено не инициализированное значение Delta '%q %s'", req.Method, req.URL)
			value0 = *result.Delta
		case http.StatusNotFound:
		default:
			dumpErr = false
			suite.T().Fatalf("Несоответствие статус кода %d ответа ожидаемому http.StatusNotFound или http.StatusOK в хендлере %q: %q", resp.StatusCode(), req.Method, req.URL)
			return
		}

		resp, err = req.
			SetContext(ctx).
			SetBody(&Metrics{
				ID:    id,
				MType: "counter",
				Delta: &value1,
			}).
			Post("update/")

		dumpErr = dumpErr && suite.Assert().NoError(err,
			"Ошибка при попытке сделать запрос с обновлением counter")
		dumpErr = dumpErr && suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере %q: %q ", req.Method, req.URL)

		resp, err = req.
			SetContext(ctx).
			SetBody(&Metrics{
				ID:    id,
				MType: "counter",
				Delta: &value2,
			}).
			Post("update/")

		dumpErr = dumpErr && suite.Assert().NoError(err,
			"Ошибка при попытке сделать запрос с обновлением counter")
		dumpErr = dumpErr && suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере %q: %q ", req.Method, req.URL)

		resp, err = req.
			SetContext(ctx).
			SetBody(&Metrics{
				ID:    id,
				MType: "counter",
			}).
			SetResult(&result).
			Post("value/")

		dumpErr = dumpErr && suite.Assert().NoError(err,
			"Ошибка при попытке сделать запрос с получением значения counter")
		dumpErr = dumpErr && suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере %q: %q ", req.Method, req.URL)
		dumpErr = dumpErr && suite.Assert().Containsf(resp.Header().Get("Content-Type"), "application/json",
			"Заголовок ответа Content-Type содержит несоответствующее значение")
		dumpErr = dumpErr && suite.NotNil(result.Delta,
			"Несоответствие отправленного значения counter (%d) полученному от сервера (nil), '%q %s'", value0+value1+value2, req.Method, req.URL)
		dumpErr = dumpErr && suite.Assert().Equalf(value0+value1+value2, *result.Delta,
			"Несоответствие отправленного значения counter (%d) полученному от сервера (%d), '%q %s'", value0+value1+value2, *result.Delta, req.Method, req.URL)

		if !dumpErr {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
			dump = dumpResponse(resp.RawResponse, true)
			suite.T().Logf("Оригинальный ответ:\n\n%s", dump)
		}

		storage = value0 + value1 + value2
	})

	suite.Run("restart server", func() {
		time.Sleep(5 * time.Second) // relax time
		suite.serverShutdown()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		suite.serverUp(ctx, suite.envs, suite.serverPort)
	})

	suite.Run("get", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		req := httpc.R().
			SetHeader("Content-Type", "application/json")

		// Вдруг на сервере уже есть значение, на всякий случай запросим.
		var result Metrics
		resp, err := req.
			SetContext(ctx).
			SetBody(&Metrics{
				ID:    id,
				MType: "counter",
			}).
			SetResult(&result).
			Post("value/")

		dumpErr := suite.Assert().NoError(err,
			"Ошибка при попытке сделать запрос с получением значения counter")

		dumpErr = dumpErr && suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере %q: %q ", req.Method, req.URL)
		dumpErr = dumpErr && suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере %q: %q ", req.Method, req.URL)
		dumpErr = dumpErr && suite.Assert().Containsf(resp.Header().Get("Content-Type"), "application/json",
			"Заголовок ответа Content-Type содержит несоответствующее значение")
		dumpErr = dumpErr && suite.NotNil(result.Delta,
			"Получено не инициализированное значение Delta '%q %s'", req.Method, req.URL)
		dumpErr = dumpErr && suite.NotNil(result.Delta,
			"Несоответствие ожидаемого значения counter (%d) полученному от сервера (nil), '%q %s'", storage, req.Method, req.URL)
		dumpErr = dumpErr && suite.Assert().Equalf(storage, *result.Delta,
			"Несоответствие ожидаемого значения counter (%d) полученному от сервера (%d), '%q %s'", storage, *result.Delta, req.Method, req.URL)

		if !dumpErr {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
			dump = dumpResponse(resp.RawResponse, true)
			suite.T().Logf("Оригинальный ответ:\n\n%s", dump)
		}
	})
}

func (suite *Iteration9Suite) TestGaugeHandlers() {
	httpc := resty.New().SetHostURL(suite.serverAddress)

	id := "GetSet" + strconv.Itoa(suite.rnd.Intn(256))
	var storage float64

	suite.Run("update", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		value := suite.rnd.Float64() * 1e6
		req := httpc.R().
			SetHeader("Content-Type", "application/json")

		resp, err := req.
			SetContext(ctx).
			SetBody(&Metrics{
				ID:    id,
				MType: "gauge",
				Value: &value,
			}).
			Post("update/")
		dumpErr := suite.Assert().NoError(err,
			"Ошибка при попытке сделать запрос с обновлением gauge")
		dumpErr = dumpErr && suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере %q: %q ", req.Method, req.URL)

		var result Metrics
		resp, err = req.
			SetContext(ctx).
			SetBody(&Metrics{
				ID:    id,
				MType: "gauge",
			}).
			SetResult(&result).
			Post("value/")

		dumpErr = dumpErr && suite.Assert().NoError(err,
			"Ошибка при попытке сделать запрос с получением значения gauge")
		dumpErr = dumpErr && suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере %q: %q ", req.Method, req.URL)
		dumpErr = dumpErr && suite.Assert().Containsf(resp.Header().Get("Content-Type"), "application/json",
			"Заголовок ответа Content-Type содержит несоответствующее значение")
		dumpErr = dumpErr && suite.Assert().NotEqualf(nil, result.Value,
			"Несоответствие отправленного значения gauge (%f) полученному от сервера (nil), '%q %s'", value, req.Method, req.URL)
		dumpErr = dumpErr && suite.Assert().Equalf(value, *result.Value,
			"Несоответствие отправленного значения gauge (%f) полученному от сервера (%f), '%q %s'", value, *result.Value, req.Method, req.URL)

		if !dumpErr {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
			dump = dumpResponse(resp.RawResponse, true)
			suite.T().Logf("Оригинальный ответ:\n\n%s", dump)
		}
		if result.Value != nil {
			storage = *result.Value
		}
	})

	suite.Run("restart server", func() {
		time.Sleep(5 * time.Second) // relax time
		suite.serverShutdown()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		suite.serverUp(ctx, suite.envs, suite.serverPort)
	})

	suite.Run("get", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		req := httpc.R().
			SetHeader("Content-Type", "application/json")

		// Вдруг на сервере уже есть значение, на всякий случай запросим.
		var result Metrics
		resp, err := req.
			SetContext(ctx).
			SetBody(&Metrics{
				ID:    id,
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
		dumpErr = dumpErr && suite.Assert().NotEqualf(nil, result.Value,
			"Несоответствие ожидаемого значения gauge (%f) полученному от сервера (nil), '%q %s'", storage, req.Method, req.URL)
		dumpErr = dumpErr && suite.Assert().Equalf(storage, *result.Value,
			"Несоответствие ожидаемого значения gauge (%f) полученному от сервера (%f), '%q %s'", storage, *result.Value, req.Method, req.URL)

		if !dumpErr {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
			dump = dumpResponse(resp.RawResponse, true)
			suite.T().Logf("Оригинальный ответ:\n\n%s", dump)
		}
	})
}
