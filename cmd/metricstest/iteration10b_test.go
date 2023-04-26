package main

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/suite"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
)

type Iteration10BSuite struct {
	suite.Suite

	serverAddress string
	serverPort    string
	serverProcess *fork.BackgroundProcess

	rnd *rand.Rand
}

func (suite *Iteration10BSuite) SetupSuite() {
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
		"STORE_INTERVAL=10s",
		"DATABASE_DSN='postgres://unknown:unknown@postgres:9999/praktikum?easter_egg_msg=you_must_prefer_this_incorrect_settings_to_those_obtained_through_arguments'",
	}...)

	serverArgs := []string{
		"-r=false",
		"-d" + flagDatabaseDSN,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	suite.serverUp(ctx, envs, serverArgs, flagServerPort)
}

func (suite *Iteration10BSuite) serverUp(ctx context.Context, envs, args []string, port string) {
	p := fork.NewBackgroundProcess(context.Background(), flagServerBinaryPath,
		fork.WithEnv(envs...),
		fork.WithArgs(args...),
	)

	err := p.Start(ctx)
	if err != nil {
		suite.T().Errorf("Невозможно запустить процесс командой %q: %s. Переменные окружения: %+v, флаги командной строки: %+v", p, err, envs, args)
		return
	}

	err = p.WaitPort(ctx, "tcp", port)
	if err != nil {
		suite.T().Logf("Не удалось дождаться пока порт %s станет доступен для запроса: %s", port, err)
		return
	}
	suite.serverProcess = p
}

func (suite *Iteration10BSuite) TearDownSuite() {
	suite.serverShutdown()
}

func (suite *Iteration10BSuite) serverShutdown() {
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

func (suite *Iteration10BSuite) TestPingHandlerWithWrongSettings() {
	httpc := resty.New().
		SetBaseURL(suite.serverAddress)

	// будет пробовать получить ответ раз в секунду
	ticker := time.NewTicker(time.Second)

	// будем дожидаться результата в течении 10 секунд
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// ожидаем ответа секунду
			rctx, rcancel := context.WithTimeout(context.Background(), time.Second)
			defer rcancel()

			resp, _ := httpc.R().
				SetContext(rctx).
				Get("/ping")
			if resp != nil && resp.StatusCode() == http.StatusOK {
				suite.T().Error(
					"Хендлер вернул положительный результат при использовании заведомо неправильных параметрах подключения к серверу." +
						"(проверьте приоритет использования параметров при указании через флаги и переменные окружения)")
				return
			}
		}
	}
}
