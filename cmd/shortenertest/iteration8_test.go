package main

// Basic imports
import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/suite"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
)

// Iteration8Suite is a suite of autotests
type Iteration8Suite struct {
	suite.Suite

	serverAddress string
	serverProcess *fork.BackgroundProcess
}

// SetupSuite bootstraps suite dependencies
func (suite *Iteration8Suite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagTargetBinaryPath, "-binary-path non-empty flag required")

	suite.serverAddress = "http://localhost:8080"

	// start server
	{
		p := fork.NewBackgroundProcess(context.Background(), flagTargetBinaryPath)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := p.Start(ctx)
		if err != nil {
			suite.T().Errorf("Невозможно запустить процесс командой %s: %s", p, err)
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
func (suite *Iteration8Suite) TearDownSuite() {
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

// TestGzipCompress attempts to:
// - generate and send random URL to shorten handler (with gzip)
// - generate and send random URL to shorten API handler (without gzip)
// - fetch original URLs by sending shorten URLs to expand handler one by one
func (suite *Iteration8Suite) TestGzipCompress() {
	originalURL := generateTestURL(suite.T())
	var shortenURLs []string

	// create HTTP client without redirects support and custom resolver
	errRedirectBlocked := errors.New("HTTP redirect blocked")
	redirPolicy := resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
		return errRedirectBlocked
	})

	httpc := resty.New().
		SetHostURL(suite.serverAddress).
		SetRedirectPolicy(redirPolicy)

	suite.Run("shorten", func() {
		// gzip request body for base shorten handler
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		_, _ = zw.Write([]byte(originalURL))
		_ = zw.Close()

		req := httpc.R().
			SetBody(buf.Bytes()).
			SetHeader("Accept-Encoding", "gzip").
			SetHeader("Content-Encoding", "gzip")
		resp, err := req.Post("/")
		if err != nil {
			dump, _ := httputil.DumpRequest(req.RawRequest, true)
			suite.Require().NoErrorf(err, "Ошибка при попытке сделать запрос для сокращения URL:\n\n %s", dump)
		}

		shortenURL := string(resp.Body())

		suite.Assert().Equalf(http.StatusCreated, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)
		suite.Assert().NoErrorf(func() error {
			_, err := url.Parse(shortenURL)
			return err
		}(), "Невозможно распарсить полученный сокращенный URL - %s : %s", shortenURL, err)

		shortenURLs = append(shortenURLs, shortenURL)
	})

	suite.Run("shorten_api", func() {
		type shortenRequest struct {
			URL string `json:"url"`
		}

		type shortenResponse struct {
			Result string `json:"result"`
		}

		var result shortenResponse

		req := httpc.R().
			SetHeader("Content-Type", "application/json").
			SetBody(&shortenRequest{
				URL: originalURL,
			}).
			SetResult(&result)
		resp, err := req.Post("/api/shorten")
		if err != nil {
			dump, _ := httputil.DumpRequest(req.RawRequest, true)
			suite.Require().NoErrorf(err, "Ошибка при попытке сделать запрос для сокращения URL:\n\n %s", dump)
		}

		shortenURL := result.Result

		suite.Assert().Equalf(http.StatusCreated, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)
		suite.Assert().NoErrorf(func() error {
			_, err := url.Parse(shortenURL)
			return err
		}(), "Невозможно распарсить полученный сокращенный URL - %s : %s", shortenURL, err)

		shortenURLs = append(shortenURLs, shortenURL)
	})

	suite.Run("expand", func() {
		for _, shortenURL := range shortenURLs {
			req := resty.New().
				SetRedirectPolicy(redirPolicy).
				R()
			resp, err := req.Get(shortenURL)
			if !errors.Is(err, errRedirectBlocked) {
				dump, _ := httputil.DumpRequest(req.RawRequest, false)
				suite.Require().NoErrorf(err, "Ошибка при попытке сделать запрос для получения исходного URL:\n\n %s", dump)
			}

			suite.Assert().Equalf(http.StatusTemporaryRedirect, resp.StatusCode(),
				"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
			)
			suite.Assert().Equalf(originalURL, resp.Header().Get("Location"),
				"Несоответствие URL полученного в заголовке Location ожидаемому",
			)
		}
	})
}
