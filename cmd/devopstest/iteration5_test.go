package main

// Basic imports
import (
	"context"
	"errors"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/suite"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
)

// Iteration5Suite is a suite of autotests
type Iteration5Suite struct {
	suite.Suite

	serverAddress string
	serverProcess *fork.BackgroundProcess
	agentProcess  *fork.BackgroundProcess
}

func (suite *Iteration5Suite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")
	suite.Require().NotEmpty(flagServerBinaryPath, "-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagAgentBinaryPath, "-agent-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagServerPort, "-server-port non-empty flag required")

	suite.serverAddress = "http://localhost:" + flagServerPort

	envs := append(os.Environ(), []string{
		"ADDRESS=" + "localhost:" + flagServerPort,
		"REPORT_INTERVAL=" + "10s",
		"POLL_INTERVAL=" + "2s",
		"RESTORE=false",

		"SHUTDOWN_TIMEOUT=" + "5s",
	}...)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	suite.agentUp(ctx, envs, flagServerPort)
	suite.serverUp(ctx, envs, flagServerPort)
}

func (suite *Iteration5Suite) serverUp(ctx context.Context, envs []string, port string) {
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

func (suite *Iteration5Suite) agentUp(ctx context.Context, envs []string, port string) {
	p := fork.NewBackgroundProcess(context.Background(), flagAgentBinaryPath,
		fork.WithEnv(envs...),
	)

	err := p.Start(ctx)
	if err != nil {
		suite.T().Errorf("Невозможно запустить процесс командой %s: %s. Переменные окружения: %+v", p, err, envs)
		return
	}

	err = p.ListenPort(ctx, "tcp", port)
	if err != nil {
		suite.T().Errorf("Не удалось дождаться пока на порт %s начнут поступать данные: %s", port, err)
		return
	}
	suite.agentProcess = p
}

// TearDownSuite teardowns suite dependencies
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

// func (suite *Iteration5Suite) TestEnvCollectAgentMetrics() {
// }

func (suite *Iteration5Suite) TestEnvCollectAgentMetrics() {
	errRedirectBlocked := errors.New("HTTP redirect blocked")
	redirPolicy := resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
		return errRedirectBlocked
	})
	httpc := resty.New().
		SetHostURL(suite.serverAddress).
		SetRedirectPolicy(redirPolicy)

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
		// {method: "gauge", name: "Alloc"},
		// {method: "gauge", name: "BuckHashSys", static: true},
		// {method: "gauge", name: "Frees"},
		// {method: "gauge", name: "GCCPUFraction", static: true},
		// {method: "gauge", name: "GCSys", static: true},
		// {method: "gauge", name: "HeapAlloc"},
		// {method: "gauge", name: "HeapIdle"},
		// {method: "gauge", name: "HeapInuse"},
		// {method: "gauge", name: "HeapObjects"},
		// {method: "gauge", name: "HeapReleased", static: true},
		// {method: "gauge", name: "HeapSys", static: true},
		// {method: "gauge", name: "LastGC", static: true},
		// {method: "gauge", name: "Lookups", static: true},
		// {method: "gauge", name: "MCacheInuse", static: true},
		// {method: "gauge", name: "MCacheSys", static: true},
		// {method: "gauge", name: "MSpanInuse", static: true},
		// {method: "gauge", name: "MSpanSys", static: true},
		// {method: "gauge", name: "Mallocs"},
		// {method: "gauge", name: "NextGC", static: true},
		// {method: "gauge", name: "NumForcedGC", static: true},
		// {method: "gauge", name: "NumGC", static: true},
		// {method: "gauge", name: "OtherSys", static: true},
		// {method: "gauge", name: "PauseTotalNs", static: true},
		// {method: "gauge", name: "StackInuse", static: true},
		// {method: "gauge", name: "StackSys", static: true},
		// {method: "gauge", name: "Sys", static: true},
		{method: "gauge", name: "TotalAlloc"},
	}

	req := httpc.R().
		SetHeader("Content-Type", "application/json")

	timer := time.NewTimer(time.Minute)

cont:
	for ok := 0; ok != len(tests); {
		// suite.T().Log("tick", len(tests)-ok)
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

			dumpErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос с получением значения %s", tt.name)

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
			suite.Assert().Truef(tt.ok, "Отсутствует изменение метрики: %s, тип: %s", tt.name, tt.method)
		})
	}
}
