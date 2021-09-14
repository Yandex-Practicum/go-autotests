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
		if err != nil {
			dump := dumpRequest(req.RawRequest, true)
			suite.Require().NoErrorf(err, "Ошибка при попытке сделать запрос для сокращения URL:\n\n %s", dump)
		}

		shortenURL = string(resp.Body())

		suite.Assert().Equalf(http.StatusCreated, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)
		suite.Assert().NoErrorf(func() error {
			_, err := url.Parse(shortenURL)
			return err
		}(), "Невозможно распарсить полученный сокращенный URL - %s : %s", shortenURL, err)
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
		resp, err := req.Get("/user/urls")
		if err != nil {
			dump := dumpRequest(req.RawRequest, false)
			suite.Require().NoErrorf(err, "Ошибка при попытке сделать запрос для получения списка сокращенных URL:\n\n %s", dump)
		}

		suite.Assert().Containsf(resp.Header().Get("Content-Type"), "application/json",
			"Заголовок ответа Content-Type содержит несоответствующее значение",
		)
		suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)

		expectedBody := []respPair{
			{
				ShortURL:    shortenURL,
				OriginalURL: originalURL,
			},
		}

		suite.Assert().Equalf(expectedBody, respBody,
			"Данные в теле ответа не соответствуют ожидаемым",
		)
	})

	suite.Run("fetch_no_urls", func() {
		req := resty.New().
			SetHostURL(suite.serverAddress).
			R()
		resp, err := req.Get("/user/urls")
		if err != nil {
			dump := dumpRequest(req.RawRequest, false)
			suite.Require().NoErrorf(err, "Ошибка при попытке сделать запрос для получения списка сокращенных URL:\n\n %s", dump)
		}

		suite.Assert().Equalf(http.StatusNoContent, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)
	})
}
