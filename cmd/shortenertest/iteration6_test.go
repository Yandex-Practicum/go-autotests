package main

// Basic imports
import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/suite"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
)

// Iteration6Suite is a suite of autotests
type Iteration6Suite struct {
	suite.Suite

	serverAddress    string
	serverProcess    *fork.BackgroundProcess
	knownPgLibraries []string
}

// SetupSuite bootstraps suite dependencies
func (suite *Iteration6Suite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagTargetBinaryPath, "-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagFileStoragePath, "-file-storage-path non-empty flag required")
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")

	suite.serverAddress = "http://localhost:8080"
	suite.knownPgLibraries = []string{
		"database/sql",
		"github.com/jackc/pgx",
		"github.com/lib/pq",
	}

	// start server
	{
		envs := append(os.Environ(), []string{
			"FILE_STORAGE_PATH=" + flagFileStoragePath,
		}...)

		p := fork.NewBackgroundProcess(context.Background(), flagTargetBinaryPath,
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
		err = p.WaitPort(ctx, "tcp", port)
		if err != nil {
			suite.T().Errorf("Не удалось дождаться пока порт %s станет доступен для запроса: %s", port, err)
			return
		}

		suite.serverProcess = p
	}
}

// TearDownSuite teardowns suite dependencies
func (suite *Iteration6Suite) TearDownSuite() {
	suite.stopServer()
}

// TestPersistentFile attempts to:
// - call handlers as Iteration1Suite.TestHandlers does
// - check if file at flagFileStoragePath is not empty
func (suite *Iteration6Suite) TestPersistentFile() {
	originalURL := generateTestURL(suite.T())
	var shortenURL string

	// create HTTP client without redirects support
	errRedirectBlocked := errors.New("HTTP redirect blocked")
	redirPolicy := resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
		return errRedirectBlocked
	})

	httpc := resty.New().
		SetHostURL(suite.serverAddress).
		SetRedirectPolicy(redirPolicy)

	suite.Run("shorten", func() {
		req := httpc.R().
			SetBody(originalURL)
		resp, err := req.Post("/")

		noRespErr := suite.Assert().NoError(err, "Ошибка при попытке сделать запрос для сокращения URL")

		shortenURL = string(resp.Body())

		validStatus := suite.Assert().Equalf(http.StatusCreated, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

		_, urlParseErr := url.Parse(shortenURL)
		validURL := suite.Assert().NoErrorf(urlParseErr,
			"Невозможно распарсить полученный сокращенный URL - %s : %s", shortenURL, err,
		)

		if !noRespErr || !validStatus || !validURL {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("expand", func() {
		req := resty.New().
			SetRedirectPolicy(redirPolicy).
			R()
		resp, err := req.Get(shortenURL)

		noRespErr := true
		if !errors.Is(err, errRedirectBlocked) {
			noRespErr = suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос для получения исходного URL")
		}

		validStatus := suite.Assert().Equalf(http.StatusTemporaryRedirect, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)
		validURL := suite.Assert().Equalf(originalURL, resp.Header().Get("Location"),
			"Несоответствие URL полученного в заголовке Location ожидаемому",
		)

		if !noRespErr || !validStatus || !validURL {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("check_file", func() {
		err := usesKnownPackage(suite.T(), flagTargetSourcePath, suite.knownPgLibraries)
		if errors.Is(err, errUsageFound) {
			suite.T().Skip("найдено использование СУБД")
			return
		}

		// stop server in case of file flushed on exit
		suite.stopServer()

		suite.Assert().FileExistsf(flagFileStoragePath, "Не удалось найти файл с сохраненными URL")
		b, err := os.ReadFile(flagFileStoragePath)
		suite.Require().NoErrorf(err, "Ошибка при чтении файла с сохраненными URL")
		suite.Assert().NotEmptyf(b, "Файл с сохраненными URL не должен быть пуст")
	})
}

// TearDownSuite teardowns suite dependencies
func (suite *Iteration6Suite) stopServer() {
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
		suite.T().Logf("Процесс завершился с не нулевым статусом: %s", err)

		// try to read stderr
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		out := suite.serverProcess.Stderr(ctx)
		if len(out) > 0 {
			suite.T().Logf("Получен лог процесса:\n\n%s", string(out))
		}

		return
	}
}
