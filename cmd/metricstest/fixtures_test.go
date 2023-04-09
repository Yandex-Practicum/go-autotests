package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
	"github.com/go-resty/resty/v2"
	"github.com/rekby/fixenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

const (
	startProcessTimeout = time.Second * 10
	checkPortInterval   = time.Millisecond * 10
)

type Env struct {
	fixenv.EnvT
	assert.Assertions
	Ctx     context.Context
	Require require.Assertions

	t testing.TB
}

func New(t testing.TB) *Env {
	ctx, ctxCancel := context.WithCancel(context.Background())
	t.Cleanup(ctxCancel)

	res := Env{
		EnvT:       *fixenv.NewEnv(t),
		Assertions: *assert.New(t),
		Require:    *require.New(t),
		t:          t,
		Ctx:        ctx,
	}
	return &res
}

func (e *Env) Errorf(format string, args ...any) {
	e.t.Helper()
	e.t.Errorf(format, args...)
}

func (e *Env) Fatalf(format string, args ...any) {
	e.t.Helper()
	e.T().Fatalf(format, args...)
}

func (e *Env) Logf(format string, args ...any) {
	e.t.Helper()
	e.t.Logf(format, args...)
}

func ExistPath(e *Env, filePath string) string {
	return fixenv.Cache(&e.EnvT, filePath, &fixenv.FixtureOptions{
		Scope: fixenv.ScopePackage,
	}, func() (string, error) {
		absFilePath, err := filepath.Abs(filePath)
		if err != nil {
			return "", err
		}
		e.Logf("Проверяю наличие файла: %q (%q)", absFilePath, filePath)

		_, err = os.Stat(filePath)
		if err != nil {
			return "", err
		}
		return filePath, nil
	})
}

func AgentFilePath(e *Env) string {
	return ExistPath(e, flagAgentBinaryPath)
}

func ServerFilePath(e *Env) string {
	return ExistPath(e, flagServerBinaryPath)
}

func ServerAddress(e *Env) string {
	return fmt.Sprintf("http://%v:%v", ServerHost(e), ServerPort(e))
}

func ServerHost(e *Env) string {
	return flagServerHost
}

func ServerPort(e *Env) int {
	return fixenv.Cache(e, nil, nil, func() (int, error) {
		return strconv.Atoi(flagServerPort)
	})
}

func AgentSourcePath(e *Env) string {
	return filepath.Join(TargetSourcePath(e), "cmd/agent")
}

func ServerSourcePath(e *Env) string {
	return filepath.Join(TargetSourcePath(e), "cmd/server")
}

func TargetSourcePath(e *Env) string {
	return fixenv.Cache(e, nil, &fixenv.FixtureOptions{Scope: fixenv.ScopePackage}, func() (string, error) {
		// project/cmd/server/server
		startPath := flagServerBinaryPath
		if startPath == "" {
			startPath = flagAgentBinaryPath
		}

		absPath, err := filepath.Abs(startPath)
		e.NoError(err, "Не могу построить полный путь к начальному файлу %q")

		// project/cmd/server
		absPath = filepath.Dir(absPath)

		// project/cmd
		absPath = filepath.Dir(absPath)

		// project
		absPath = filepath.Dir(absPath)

		e.Logf("Проверяю что %q (%q) - это папка", absPath, flagTargetSourcePath)
		stat, err := os.Stat(absPath)
		if err != nil {
			e.Fatalf("Не могу получить информацию о папке", err)
			return "", err
		}

		if stat.IsDir() {
			return absPath, nil
		}

		return "", fmt.Errorf("%q - не папка", absPath)
	})
}

func StartProcess(e *Env, name string, command string, args ...string) *fork.BackgroundProcess {
	cacheKey := append([]string{name, command}, args...)
	return fixenv.CacheWithCleanup(e, cacheKey, nil, func() (*fork.BackgroundProcess, fixenv.FixtureCleanupFunc, error) {
		res := fork.NewBackgroundProcess(e.Ctx, command, fork.WithArgs(args...))

		e.Logf("Запускаю %q: %q %#v", name, command, args)
		err := res.Start(e.Ctx)
		if err != nil {
			return nil, nil, err
		}

		cleanup := func() {
			e.Logf("Останавливаю %q: %q %#v", name, command, args)
			exitCode, stopErr := res.Stop(syscall.SIGINT, syscall.SIGKILL)

			stdOut := string(res.Stdout(context.Background()))
			stdErr := string(res.Stderr(context.Background()))

			e.Logf("stdout:\n%v", stdOut)
			e.Logf("stderr:\n%v", stdErr)

			if stopErr != nil {
				e.Fatalf("Не получилось остановить процесс: %+v", stopErr)
			}
			if exitCode > 0 {
				e.Logf("Ненулевой код возврата: %v", exitCode)
			}
		}
		return res, cleanup, nil
	})
}

func StartProcessWhichListenPort(e *Env, host string, port int, name string, command string, args ...string) *fork.BackgroundProcess {
	cacheKey := append([]string{host, strconv.Itoa(port), name, command}, args...)
	return fixenv.Cache[*fork.BackgroundProcess](e, cacheKey, nil, func() (*fork.BackgroundProcess, error) {
		process := StartProcess(e, name, command, args...)
		address := fmt.Sprintf("%v:%v", host, port)
		return process, waitOpenPort(e, address)
	})
}

func ServerMock(e *Env, port int) *TestServerT {
	return fixenv.CacheWithCleanup(e, port, nil, func() (*TestServerT, fixenv.FixtureCleanupFunc, error) {
		endpoint := "localhost:" + strconv.Itoa(port)
		res := NewTestServerT(e, endpoint)

		e.Logf("Запускаю мок сервер: %q", endpoint)
		go res.Start()

		err := waitOpenPort(e, endpoint)

		return res, res.Stop, err
	})
}

func RestyClient(e *Env, host string) *resty.Client {
	return fixenv.Cache(e, host, nil, func() (*resty.Client, error) {
		return resty.
			New().
			SetDebug(true).
			SetBaseURL(host).
			SetRedirectPolicy(resty.NoRedirectPolicy()).
			SetLogger(restyLogger{e}), nil
	})
}

type restyLogger struct {
	e *Env
}

func (l restyLogger) Errorf(format string, v ...interface{}) {
	l.e.Logf("RESTY ERROR: "+format, v...)
}

func (l restyLogger) Warnf(format string, v ...interface{}) {
	l.e.Logf("resty warn: "+format, v...)
}

func (l restyLogger) Debugf(format string, v ...interface{}) {
	l.e.Logf("resty: "+format, v...)
}

func waitOpenPort(e *Env, address string) error {
	ctx, cancel := context.WithTimeout(e.Ctx, startProcessTimeout)
	defer cancel()

	dialer := net.Dialer{}
	e.Logf("Пробую подключиться на %q...", address)

	for {
		time.Sleep(checkPortInterval)
		conn, err := dialer.DialContext(ctx, "tcp", address)
		if err == nil {
			e.Logf("Закрываю успешное подключение")
			err = conn.Close()
			return err
		}
		if ctx.Err() != nil {
			e.Fatalf("Ошибка подлючения: %+v", err)
			return err
		}
	}
}
