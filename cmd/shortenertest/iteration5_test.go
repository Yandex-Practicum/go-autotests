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

// Iteration5Suite is a suite of autotests
type Iteration5Suite struct {
	suite.Suite

	serverAddress string
	serverBaseURL string
	serverProcess *fork.BackgroundProcess
}

// SetupSuite bootstraps suite dependencies
func (suite *Iteration5Suite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagTargetBinaryPath, "-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagServerPort, "-server-port non-empty flag required")

	// start server
	{
		suite.serverAddress = "localhost:" + flagServerPort
		suite.serverBaseURL = "http://" + suite.serverAddress
		envs := append(os.Environ(), []string{
			"SERVER_ADDRESS=" + suite.serverAddress,
			"BASE_URL=" + suite.serverBaseURL,
		}...)

		p := fork.NewBackgroundProcess(context.Background(), flagTargetBinaryPath,
			fork.WithEnv(envs...),
		)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := p.Start(ctx)
		if err != nil {
			suite.T().Errorf("Невозможно запустить процесс командой %s: %s. Переменные окружения: %+v", p, err, envs)
			return
		}

		err = p.WaitPort(ctx, "tcp", flagServerPort)
		if err != nil {
			suite.T().Errorf("Не удалось дождаться пока порт %s станет доступен для запроса: %s", flagServerPort, err)
			return
		}

		suite.serverProcess = p
	}
}

// TearDownSuite teardowns suite dependencies
func (suite *Iteration5Suite) TearDownSuite() {
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

// TestEnvVars attempts to:
// - generate and send random URL to shorten handler
// - generate and send random URL to shorten API handler
// - fetch original URLs by sending shorten URLs to expand handler one by one
func (suite *Iteration5Suite) TestEnvVars() {
	originalURL := generateTestURL(suite.T())
	var shortenURLs []string

	// create HTTP client without redirects support and custom resolver
	errRedirectBlocked := errors.New("HTTP redirect blocked")
	redirPolicy := resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
		return errRedirectBlocked
	})

	restyClient := resty.New()
	transport := restyClient.GetClient().Transport.(*http.Transport)

	// mock all network requests to be resolved at localhost
	resolveIP := "127.0.0.1:" + flagServerPort
	transport.DialContext = mockResolver("tcp", suite.serverAddress, resolveIP)

	httpc := restyClient.
		SetTransport(transport).
		SetHostURL(suite.serverBaseURL).
		SetRedirectPolicy(redirPolicy)

	suite.Run("shorten", func() {
		req := httpc.R().
			SetBody(originalURL)
		resp, err := req.Post("/")
		if err != nil {
			dump := dumpRequest(req.RawRequest, true)
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
			dump := dumpRequest(req.RawRequest, true)
			suite.Require().NoErrorf(err, "Ошибка при попытке сделать запрос для сокращения URL:\n\n %s", dump)
		}

		shortenURL := result.Result

		suite.Assert().Containsf(resp.Header().Get("Content-Type"), "application/json",
			"Заголовок ответа Content-Type содержит несоответствующее значение",
		)
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
				dump := dumpRequest(req.RawRequest, false)
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
