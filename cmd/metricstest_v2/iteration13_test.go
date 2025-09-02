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

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/suite"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
)

type Iteration13Suite struct {
	suite.Suite

	serverAddress string
	serverPort    string
	serverProcess *fork.BackgroundProcess
	agentProcess  *fork.BackgroundProcess

	rnd *rand.Rand
}

func (suite *Iteration13Suite) SetupSuite() {
	// check required flags
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

	agentArgs := []string{
		"-p=1",
	}
	serverArgs := []string{
		"-r=false",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	suite.agentUp(ctx, envs, agentArgs, flagServerPort)
	suite.serverUp(ctx, envs, serverArgs, flagServerPort)
}

func (suite *Iteration13Suite) serverUp(ctx context.Context, envs, args []string, port string) {
	suite.serverProcess = fork.NewBackgroundProcess(context.Background(), flagServerBinaryPath,
		fork.WithEnv(envs...),
		fork.WithArgs(args...),
	)

	err := suite.serverProcess.Start(ctx)
	if err != nil {
		suite.T().Errorf("Невозможно запустить процесс командой %q: %s. Переменные окружения: %+v, флаги командной строки: %+v", suite.serverProcess, err, envs, args)
		return
	}

	err = suite.serverProcess.WaitPort(ctx, "tcp", port)
	if err != nil {
		suite.T().Errorf("Не удалось дождаться пока порт %s станет доступен для запроса: %s", port, err)
		return
	}
}

func (suite *Iteration13Suite) agentUp(ctx context.Context, envs, args []string, port string) {
	suite.agentProcess = fork.NewBackgroundProcess(context.Background(), flagAgentBinaryPath,
		fork.WithEnv(envs...),
		fork.WithArgs(args...),
	)

	err := suite.agentProcess.Start(ctx)
	if err != nil {
		suite.T().Errorf("Невозможно запустить процесс командой %q: %s. Переменные окружения: %+v, флаги командной строки: %+v", suite.agentProcess, err, envs, args)
		return
	}

	err = suite.agentProcess.ListenPort(ctx, "tcp", port)
	if err != nil {
		suite.T().Errorf("Не удалось дождаться пока на порт %s начнут поступать данные: %s", port, err)
		return
	}
}

func (suite *Iteration13Suite) TearDownSuite() {
	suite.agentShutdown()
	suite.serverShutdown()
}

func (suite *Iteration13Suite) serverShutdown() {
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

func (suite *Iteration13Suite) agentShutdown() {
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

	// try to read stdout/stderr
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

func (suite *Iteration13Suite) TestCounterGzipHandlers() {
	httpc := resty.New().SetHostURL(suite.serverAddress)

	id := "GetSetZip" + strconv.Itoa(suite.rnd.Intn(256))

	suite.Run("update", func() {
		value1, value2 := int64(suite.rnd.Int31()), int64(suite.rnd.Int31())
		req := httpc.R().
			SetHeader("Accept-Encoding", "gzip").
			SetHeader("Content-Type", "application/json")

		var result Metrics
		resp, err := req.
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

		resp, err = req.SetBody(
			&Metrics{
				ID:    id,
				MType: "counter",
				Delta: &value1,
			}).Post("update/")

		dumpErr = dumpErr && suite.Assert().NoError(err,
			"Ошибка при попытке сделать запрос с обновлением counter")
		dumpErr = dumpErr && suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере %q: %q ", req.Method, req.URL)

		resp, err = req.SetBody(
			&Metrics{
				ID:    id,
				MType: "counter",
				Delta: &value2,
			}).Post("update/")

		dumpErr = dumpErr && suite.Assert().NoError(err,
			"Ошибка при попытке сделать запрос с обновлением counter")
		dumpErr = dumpErr && suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере %q: %q ", req.Method, req.URL)

		resp, err = req.
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
		dumpErr = dumpErr && suite.Assert().Containsf(resp.Header().Get("Content-Encoding"), "gzip",
			"Заголовок ответа Content-Encoding содержит несоответствующее значение")
		dumpErr = dumpErr && suite.NotNil(result.Delta,
			"Несоответствие отправленного значения counter (r:%d+w:%d+w:%d) полученному от сервера (nil), '%q %s'", value0, value1, value2, req.Method, req.URL)
		dumpErr = dumpErr && suite.Assert().Equalf(value0+value1+value2, *result.Delta,
			"Несоответствие отправленного значения counter (r:%d+w:%d+w:%d) полученному от сервера (%d), '%q %s'", value0, value1, value2, *result.Delta, req.Method, req.URL)

		if !dumpErr {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
			dump = dumpResponse(resp.RawResponse, true)
			suite.T().Logf("Оригинальный ответ:\n\n%s", dump)
		}
	})
}

func (suite *Iteration13Suite) TestGaugeGzipHandlers() {
	httpc := resty.New().SetHostURL(suite.serverAddress)

	id := "GetSetZip" + strconv.Itoa(suite.rnd.Intn(256))

	suite.Run("update", func() {
		value := suite.rnd.Float64() * 1e6
		req := httpc.R().
			SetHeader("Accept-Encoding", "gzip").
			SetHeader("Content-Type", "application/json")

		resp, err := req.SetBody(
			&Metrics{
				ID:    id,
				MType: "gauge",
				Value: &value,
			}).Post("update/")

		dumpErr := suite.Assert().NoError(err,
			"Ошибка при попытке сделать запрос с обновлением gauge")
		dumpErr = dumpErr && suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере %q: %q ", req.Method, req.URL)

		var result Metrics
		resp, err = req.
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
		dumpErr = dumpErr && suite.Assert().Containsf(resp.Header().Get("Content-Encoding"), "gzip",
			"Заголовок ответа Content-Encoding содержит несоответствующее значение")
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
	})
}

func (suite *Iteration13Suite) TestCollectAgentMetrics() {
	httpc := resty.New().SetHostURL(suite.serverAddress)

	tests := []struct {
		name   string
		method string
		value  float64
		delta  int64
		update int
		ok     bool
		static bool
	}{
		{method: "counter", name: "PollCount"},
		{method: "gauge", name: "RandomValue"},
		{method: "gauge", name: "Alloc"},
		{method: "gauge", name: "BuckHashSys", static: true},
		{method: "gauge", name: "Frees"},
		{method: "gauge", name: "GCCPUFraction", static: true},
		{method: "gauge", name: "GCSys", static: true},
		{method: "gauge", name: "HeapAlloc"},
		{method: "gauge", name: "HeapIdle"},
		{method: "gauge", name: "HeapInuse"},
		{method: "gauge", name: "HeapObjects"},
		{method: "gauge", name: "HeapReleased", static: true},
		{method: "gauge", name: "HeapSys", static: true},
		{method: "gauge", name: "LastGC", static: true},
		{method: "gauge", name: "Lookups", static: true},
		{method: "gauge", name: "MCacheInuse", static: true},
		{method: "gauge", name: "MCacheSys", static: true},
		{method: "gauge", name: "MSpanInuse", static: true},
		{method: "gauge", name: "MSpanSys", static: true},
		{method: "gauge", name: "Mallocs"},
		{method: "gauge", name: "NextGC", static: true},
		{method: "gauge", name: "NumForcedGC", static: true},
		{method: "gauge", name: "NumGC", static: true},
		{method: "gauge", name: "OtherSys", static: true},
		{method: "gauge", name: "PauseTotalNs", static: true},
		{method: "gauge", name: "StackInuse", static: true},
		{method: "gauge", name: "StackSys", static: true},
		{method: "gauge", name: "Sys", static: true},
		{method: "gauge", name: "TotalAlloc"},
	}

	req := httpc.R().
		SetHeader("Content-Type", "application/json")

	timer := time.NewTimer(time.Minute)

cont:
	for ok := 0; ok != len(tests); {
		select {
		case <-timer.C:
			break cont
		default:
		}
		for i, tt := range tests {
			if tt.ok {
				continue
			}
			var (
				resp *resty.Response
				err  error
			)
			time.Sleep(100 * time.Millisecond)

			var result Metrics
			resp, err = req.
				SetBody(&Metrics{
					ID:    tt.name,
					MType: tt.method,
				}).
				SetResult(&result).
				Post("/value/")

			dumpErr := suite.Assert().NoErrorf(err,
				"Ошибка при попытке сделать запрос с получением значения %s", tt.name)

			if resp.StatusCode() == http.StatusNotFound {
				continue
			}

			dumpErr = dumpErr && suite.Assert().Containsf(resp.Header().Get("Content-Type"), "application/json",
				"Заголовок ответа Content-Type содержит несоответствующее значение")
			dumpErr = dumpErr && suite.Assert().True(result.MType != "gauge" || result.Value != nil,
				"Получен не однозначный результат (возвращаемое значение value=nil не соответствет типу gauge) '%q %s'", req.Method, req.URL)
			dumpErr = dumpErr && suite.Assert().True(result.MType != "counter" || result.Delta != nil,
				"Получен не однозначный результат (возвращаемое значение delta=nil не соответствет типу counter) '%q %s'", req.Method, req.URL)
			dumpErr = dumpErr && suite.Assert().False(result.Delta == nil && result.Value == nil,
				"Получен результат без данных (Dalta == nil && Value == nil) '%q %s'", req.Method, req.URL)
			dumpErr = dumpErr && suite.Assert().False(result.Delta != nil && result.Value != nil,
				"Получен не однозначный результат (Dalta != nil && Value != nil) '%q %s'", req.Method, req.URL)
			dumpErr = dumpErr && suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
				"Несоответствие статус кода ответа ожидаемому в хендлере %q: %q ", req.Method, req.URL)
			dumpErr = dumpErr && suite.Assert().True(result.MType == "gauge" || result.MType == "counter",
				"Получен ответ с неизвестным значением типа: %q, '%q %s'", result.MType, req.Method, req.URL)

			if !dumpErr {
				dump := dumpRequest(req.RawRequest, true)
				suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
				dump = dumpResponse(resp.RawResponse, true)
				suite.T().Logf("Оригинальный ответ:\n\n%s", dump)
				return
			}

			switch tt.method {
			case "gauge":
				if (tt.update != 0 && *result.Value != tt.value) || tt.static {
					tests[i].ok = true
					ok++
					suite.T().Logf("get %s: %q, value: %f", tt.method, tt.name, *result.Value)
				}
				tests[i].value = *result.Value
			case "counter":
				if (tt.update != 0 && *result.Delta != tt.delta) || tt.static {
					tests[i].ok = true
					ok++
					suite.T().Logf("get %s: %q, value: %d", tt.method, tt.name, *result.Delta)
				}
				tests[i].delta = *result.Delta
			}

			tests[i].update++
		}
	}
	for _, tt := range tests {
		suite.Run(tt.method+"/"+tt.name, func() {
			suite.Assert().Truef(tt.ok,
				"Отсутствует изменение метрики: %s, тип: %s", tt.name, tt.method)
		})
	}
}
