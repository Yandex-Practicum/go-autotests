package main

import (
	"context"
	"errors"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
)

func TestIteration2A(t *testing.T) {
	e := New(t)
	serverMock := ServerMock(e, serverDefaultPort)

	gauges := []string{
		"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys", "HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased", "HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys", "MSpanInuse", "MSpanSys", "Mallocs", "NextGC",
		"NumForcedGC", "NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys", "Sys", "TotalAlloc", "RandomValue",
	}
	counters := []string{
		"PollCount",
	}

	StartDefaultAgent(e)

	firstIterationTimeout := agentDefaultReportInterval + agentDefaultReportInterval/2

	e.Logf("Жду %v", firstIterationTimeout)
	time.Sleep(firstIterationTimeout)

	serverMock.CheckReceiveValues(gauges, counters, 1, 2)
	firstRandom := serverMock.GetLastGauge("RandomValue")
	e.InDelta(int(agentDefaultReportInterval/agentDefaultPollInterval), serverMock.GetLastCounter("PollCount"), 1)

	e.Logf("Жду ещё %v", agentDefaultReportInterval)
	time.Sleep(agentDefaultReportInterval)

	serverMock.CheckReceiveValues(gauges, counters, 2, 3)
	e.InDelta(agentDefaultReportInterval/agentDefaultPollInterval*2, serverMock.GetLastCounter("PollCount"), 1)
	e.NotEqual(firstRandom, serverMock.GetLastGauge("RandomValue"))
}

func StartDefaultAgent(e *Env) {
	StartProcess(e, "agent", AgentFilePath(e))
}

type Iteration2ASuite struct {
	suite.Suite

	agentAddress string
	agentProcess *fork.BackgroundProcess
}

func (suite *Iteration2ASuite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagAgentBinaryPath, "-agent-binary-path non-empty flag required")

	suite.agentAddress = "http://localhost:8080"

	envs := append(os.Environ(), []string{
		"RESTORE=false",
	}...)
	p := fork.NewBackgroundProcess(context.Background(), flagAgentBinaryPath,
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
	err = p.ListenPort(ctx, "tcp", port)
	if err != nil {
		suite.T().Errorf("Не удалось дождаться пока на порт %s начнут поступать данные: %s", port, err)
		return
	}

	suite.agentProcess = p
}

func (suite *Iteration2ASuite) TearDownSuite() {
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

// TestAgent проверяет
// агент успешно стартует и передает какие-то данные по tcp, на 127.0.0.1:8080
func (suite *Iteration2ASuite) TestAgent() {
	suite.Run("receive data from agent", func() {
	})
}
