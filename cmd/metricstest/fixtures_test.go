package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
	"github.com/go-resty/resty/v2"
	"github.com/rekby/fixenv"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

const (
	startProcessTimeout = time.Second * 10
	checkPortInterval   = time.Millisecond * 100
)

type Env struct {
	fixenv.EnvT
	assert.Assertions
	Ctx context.Context

	t testing.TB
}

func New(t testing.TB) *Env {
	ctx, ctxCancel := context.WithCancel(context.Background())
	t.Cleanup(ctxCancel)

	res := Env{
		EnvT:       *fixenv.NewEnv(t),
		Assertions: *assert.New(t),
		t:          t,
		Ctx:        ctx,
	}
	return &res
}

func (e *Env) Fatalf(format string, args ...any) {
	e.T().Fatalf(format, args...)
}

func (e *Env) Logf(format string, args ...any) {
	e.t.Logf(format, args...)
}

func

func ExistPath(e *Env, filePath string) string {
	return fixenv.Cache(&e.EnvT, filePath, &fixenv.FixtureOptions{
		Scope: fixenv.ScopePackage,
	}, func() (string, error) {
		e.Logf("Проверяю наличие файла: %q", filePath)
		_, err := os.Stat(filePath)
		if err != nil {
			return "", err
		}
		return filePath, nil
	})
}

func AgentPath(e *Env) string {
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
	return flagServerPort
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
			exitCode, err := res.Stop()
			if err != nil {
				e.Fatalf("Не получилось остановить процесс: %+v", err)
			}
			if exitCode != 0 {
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
		ctx, cancel := context.WithTimeout(e.Ctx, startProcessTimeout)
		defer cancel()

		address := fmt.Sprintf("%v:%v", host, port)
		dialer := net.Dialer{}
		for {
			time.Sleep(checkPortInterval)
			e.Logf("Пробую подключиться на %q...", address)
			conn, err := dialer.DialContext(ctx, "tcp", address)
			if err == nil {
				e.Logf("Закрываю успешное подключение")
				err = conn.Close()
				return process, err
			}
			if ctx.Err() != nil {
				return nil, err
			}
		}
	})
}

func RestyClient(e *Env, host string) *resty.Client {
	return fixenv.Cache[*resty.Client](e, host, nil, func() (*resty.Client, error) {
		return resty.New().SetHostURL(host).SetRedirectPolicy(resty.NoRedirectPolicy()), nil
	})
}
