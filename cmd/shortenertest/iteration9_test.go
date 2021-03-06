package main

// Basic imports
import (
	"context"
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/suite"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
)

// Iteration9Suite is a suite of autotests
type Iteration9Suite struct {
	suite.Suite

	serverAddress string
	serverProcess *fork.BackgroundProcess
}

// SetupSuite bootstraps suite dependencies
func (suite *Iteration9Suite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagTargetBinaryPath, "-binary-path non-empty flag required")

	suite.serverAddress = "http://localhost:8080"

	// start server
	{
		envs := os.Environ()
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
func (suite *Iteration9Suite) TearDownSuite() {
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

// TestAuth attempts to:
// - generate and send random URL to shorten handler
// - fetch URLs based on given cookie from previous handler response
func (suite *Iteration9Suite) TestAuth() {
	originalURL := generateTestURL(suite.T())
	var shortenURL string

	jar, err := cookiejar.New(nil)
	suite.Require().NoError(err, "Неожиданная ошибка при создании Cookie Jar")

	httpc := resty.New().
		SetHostURL(suite.serverAddress).
		SetCookieJar(jar)

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

	suite.Run("fetch_urls", func() {
		type respPair struct {
			ShortURL    string `json:"short_url"`
			OriginalURL string `json:"original_url"`
		}

		var respBody []respPair

		req := httpc.R().
			SetHeader("Accept-Encoding", "identity").
			SetResult(&respBody)
		resp, err := req.Get("/api/user/urls")

		noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос для получения списка сокращенных URL")

		validContentType := suite.Assert().Containsf(resp.Header().Get("Content-Type"), "application/json",
			"Заголовок ответа Content-Type содержит несоответствующее значение",
		)
		validStatus := suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)

		expectedBody := []respPair{
			{
				ShortURL:    shortenURL,
				OriginalURL: originalURL,
			},
		}

		validBody := suite.Assert().Equalf(expectedBody, respBody,
			"Данные в теле ответа не соответствуют ожидаемым",
		)

		if !noRespErr || !validStatus || !validContentType || !validBody {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("fetch_no_urls", func() {
		req := resty.New().
			SetHostURL(suite.serverAddress).
			R()
		resp, err := req.Get("/api/user/urls")
		if err != nil {
			dump := dumpRequest(req.RawRequest, false)
			suite.Require().NoErrorf(err, "Ошибка при попытке сделать запрос для получения списка сокращенных URL:\n\n %s", dump)
		}

		suite.Assert().Equalf(http.StatusNoContent, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)
	})
}
