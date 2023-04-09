package main

import (
	"testing"
	"time"
)

func TestIteration2A(t *testing.T) {
	e := New(t)
	ServerPort(e, serverDefaultPort)
	AgentReportInterval(e, agentDefaultReportInterval)
	AgentPollInterval(e, agentDefaultPollInterval)
	testAgentIncrement2(e, StartDefaultAgent)
}

func StartDefaultAgent(e *Env) {
	StartProcess(e, "agent", AgentFilePath(e))
}

func testAgentIncrement2(e *Env, startAgent func(e *Env)) {
	e.Logf("Тестирование функционала второго инкремента")

	serverMock := ServerMock(e, ServerPort(e))

	gauges := []string{
		"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys", "HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased", "HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys", "MSpanInuse", "MSpanSys", "Mallocs", "NextGC",
		"NumForcedGC", "NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys", "Sys", "TotalAlloc", "RandomValue",
	}
	counters := []string{
		"PollCount",
	}

	startAgent(e)

	reportInterval := AgentReportInterval(e)
	pollInterval := AgentPollInterval(e)
	firstIterationTimeout := reportInterval + reportInterval/2

	e.Logf("Жду %v", firstIterationTimeout)
	time.Sleep(firstIterationTimeout)

	serverMock.CheckReceiveValues(gauges, counters, 1, 2)
	firstRandom := serverMock.GetLastGauge("RandomValue")
	e.InDelta(int(reportInterval/pollInterval), serverMock.GetLastCounter("PollCount"), 1)

	e.Logf("Жду ещё %v", reportInterval)
	time.Sleep(reportInterval)

	serverMock.CheckReceiveValues(gauges, counters, 2, 3)
	e.InDelta(reportInterval/pollInterval*2, serverMock.GetLastCounter("PollCount"), 1)
	e.NotEqual(firstRandom, serverMock.GetLastGauge("RandomValue"), "Случайное значение не поменялось при повторной отправке")
}
