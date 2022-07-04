package main

// Basic imports
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

// Iteration10Suite is a suite of autotests
type Iteration10Suite struct {
	suite.Suite

	serverAddress  string
	serverPort     string
	serverProcess  *fork.BackgroundProcess
	serverArgs     []string
	knownLibraries []string

	rnd  *rand.Rand
	envs []string
	key  []byte
}

func (suite *Iteration10Suite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")
	suite.Require().NotEmpty(flagServerBinaryPath, "-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagAgentBinaryPath, "-agent-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagServerPort, "-server-port non-empty flag required")
	suite.Require().NotEmpty(flagSHA256Key, "-key non-empty flag required")
	suite.Require().NotEmpty(flagDatabaseDSN, "-database-dsn non-empty flag required")

	suite.rnd = rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
	suite.serverAddress = "http://localhost:" + flagServerPort
	suite.serverPort = flagServerPort

	suite.key = []byte(flagSHA256Key)

	suite.envs = append(os.Environ(), []string{
		"RESTORE=true",
		// "KEY=" + flagSHA256Key,
	}...)

	suite.serverArgs = []string{
		"-a=localhost:" + flagServerPort,
		// "-s=5s",
		"-r=false",
		"-i=5m",
		"-k=" + flagSHA256Key,
		"-d=" + flagDatabaseDSN,
	}

	suite.knownLibraries = []string{
		"database/sql",
		"github.com/jackc/pgx",
		"github.com/lib/pq",
		"github.com/jmoiron/sqlx",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	suite.serverUp(ctx, suite.envs, suite.serverArgs, flagServerPort)
}

func (suite *Iteration10Suite) serverUp(ctx context.Context, envs, args []string, port string) {
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
		suite.T().Errorf("Не удалось дождаться пока порт %s станет доступен для запроса: %s", port, err)
		return
	}
	suite.serverProcess = p
}

// TearDownSuite teardowns suite dependencies
func (suite *Iteration10Suite) TearDownSuite() {
	suite.serverShutdown()
}

func (suite *Iteration10Suite) serverShutdown() {
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

// TestLibraryUsage attempts to recursively find usage of database/sql in given sources
func (suite *Iteration10Suite) TestDBLibraryUsage() {
	err := usesKnownPackage(suite.T(), flagTargetSourcePath, suite.knownLibraries)
	if errors.Is(err, errUsageFound) {
		return
	}
	if err == nil || errors.Is(err, errUsageNotFound) {
		suite.T().Errorf("Не найдено использование библиотеки database/sql по пути %q", flagTargetSourcePath)
		return
	}
	suite.T().Errorf("Неожиданная ошибка при поиске использования библиотеки по пути %q, %v", flagTargetSourcePath, err)
}

// TestPingHandler attempts to call for ping handler and check positive result
func (suite *Iteration10Suite) TestPingHandler() {
	httpc := resty.New().
		SetHostURL(suite.serverAddress)

	ticker := time.NewTicker(time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			suite.T().Error("Не удалось получить код ответа 200 от хендлера 'GET /ping' за отведенное время")
			return
		case <-ticker.C:
			resp, _ := httpc.R().Get("/ping")
			if resp != nil && resp.StatusCode() == http.StatusOK {
				return
			}
		}
	}
}
