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
	"github.com/Yandex-Practicum/go-autotests/internal/random"
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

	t *testing.T
}

func New(t *testing.T) *Env {
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

func (e *Env) Test() *testing.T {
	return e.t
}

///
/// В этих тестах используется библиотека fixenv. Она помогает создавать тестовое окружение.
/// Фикстура - это функция, которая может выполнять фоновую работу, возвращает значение и может выполнять очистку за собой после завершения теста.
/// Вызовы фикстуры идемпотентны, т.е. многократный вызов фикстуры с одними и теми же параметрами внутри одного окружения всегда будет возвращать один и тот же результат.
/// при этом фоновая работа выполняется только при первом вызове.
///
/// Env - тестовое окружение, которое создаётся для каждого теста.
/// Подробнее о билиотеке можно почитать на странице https://github.com/rekby/fixenv
///

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

func AgentPollInterval(e *Env, setInterval ...time.Duration) time.Duration {
	return fixenv.Cache(e, "", &fixenv.FixtureOptions{Scope: fixenv.ScopeTestAndSubtests}, func() (time.Duration, error) {
		switch len(setInterval) {
		case 0:
			return time.Second, nil
		case 1:
			return setInterval[0], nil
		default:
			return 0, fmt.Errorf("В опциональном параметре можно передать максимум одно значение")
		}
	})
}

func AgentReportInterval(e *Env, setInterval ...time.Duration) time.Duration {
	return fixenv.Cache(e, "", &fixenv.FixtureOptions{Scope: fixenv.ScopeTestAndSubtests}, func() (time.Duration, error) {
		switch len(setInterval) {
		case 0:
			return 2 * time.Second, nil
		case 1:
			return setInterval[0], nil
		default:
			return 0, fmt.Errorf("В опциональном параметре можно передать максимум одно значение")
		}
	})
}

func ServerFilePath(e *Env) string {
	return ExistPath(e, flagServerBinaryPath)
}

func ConnectToServer(e *Env) *resty.Client {
	return RestyClient(e, "http://"+ServerAddress(e))
}

func ServerAddress(e *Env) string {
	return fixenv.Cache(e, "", nil, func() (string, error) {
		res := fmt.Sprintf("%v:%v", ServerHost(e), ServerPort(e))
		e.Logf("Адрес сервера: %q", res)
		return res, nil
	})
}

func ServerHost(e *Env) string {
	return "localhost"
}

func ServerPort(e *Env, setPort ...int) int {
	return fixenv.Cache(e, "", &fixenv.FixtureOptions{Scope: fixenv.ScopeTestAndSubtests}, func() (int, error) {
		port := 0
		var err error
		switch len(setPort) {
		case 0:
			e.Logf("Автоматический выбор серверного порта")
			port, err = random.UnusedPort()
			if err != nil {
				return 0, err
			}
		case 1:
			port = setPort[0]
			e.Logf("Серверный порт задан вручную")
		default:
			return 0, fmt.Errorf("В опциональном параметре можно передать максимум одно значение")
		}
		e.Logf("Для сервера выбран порт: %v", port)
		return port, err
	})
}

func AgentSourcePath(e *Env) string {
	return fixenv.Cache(e, nil, &fixenv.FixtureOptions{Scope: fixenv.ScopePackage}, func() (string, error) {
		res := filepath.Join(TargetSourcePath(e), "cmd/agent")
		e.Logf("Путь к исходникам агента: %q", res)
		e.DirExists(res)
		return res, nil
	})
}

func ServerSourcePath(e *Env) string {
	return fixenv.Cache(e, nil, &fixenv.FixtureOptions{Scope: fixenv.ScopePackage}, func() (string, error) {
		res := filepath.Join(TargetSourcePath(e), "cmd/server")
		e.Logf("Путь к исходникам сервера: %q", res)
		e.DirExists(res)
		return res, nil
	})
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
	return fixenv.Cache(e, cacheKey, nil, func() (*fork.BackgroundProcess, error) {
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
		go func() { _ = res.Start() }()

		err := waitOpenPort(e, endpoint)

		return res, res.Stop, err
	})
}

func RestyClient(e *Env, baseUrl string) *resty.Client {
	return fixenv.Cache(e, baseUrl, nil, func() (*resty.Client, error) {
		e.Logf("Создаётся клиент для %q", baseUrl)
		return resty.
			New().
			SetDebug(true).
			SetBaseURL(baseUrl).
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
