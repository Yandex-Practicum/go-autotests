package main

import (
	"fmt"
	"testing"
	"time"
)

func TestIteration4(t *testing.T) {
	_ = New(t)
	t.Run("server", func(t *testing.T) {
		testServerIncrement1(t, StartServerWithArgs)
		testServerIncrement3(t, StartServerWithArgs)
	})

	t.Run("agent", func(t *testing.T) {
		e := New(t)
		testAgentIncrement2(e, StartAgentWithArgs)
	})
}

func StartServerWithArgs(e *Env) string {
	serverArg := "-a=" + ServerAddress(e)
	StartProcessWhichListenPort(e, ServerHost(e), ServerPort(e), "server", ServerFilePath(e), serverArg)
	return "http://" + ServerAddress(e)
}

func StartAgentWithArgs(e *Env) {
	serverArg := "-a=" + ServerAddress(e)
	pollIntervalArg := fmt.Sprintf("-p=%v", int(AgentPollInterval(e)/time.Second))
	reportIntervalArg := fmt.Sprintf("-r=%v", int(AgentReportInterval(e)/time.Second))
	StartProcess(e, "agent", AgentFilePath(e), serverArg, pollIntervalArg, reportIntervalArg)
}
