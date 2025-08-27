package main

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

// Iteration7Suite является сьютом с тестами и состоянием для инкремента
type Iteration4Suite struct {
	suite.Suite

	serverAddress    string
	serverBaseURL    string
	serverProcess    *fork.BackgroundProcess
	knownPgLibraries []string
}

// SetupSuite подготавливает необходимые зависимости
func (suite *Iteration4Suite) SetupSuite() {
	// проверяем наличие необходимых флагов
	suite.Require().NotEmpty(flagTargetBinaryPath, "-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagServerPort, "-server-port non-empty flag required")

	// запускаем процесс тестируемого сервера
	{
		suite.serverAddress = "localhost:" + flagServerPort
		suite.serverBaseURL = "http://" + suite.serverAddress

		// передаем флаги в процесс сервера
		args := []string{
			"-a=" + suite.serverAddress,
			"-b=" + suite.serverBaseURL,
		}

		p := fork.NewBackgroundProcess(context.Background(), flagTargetBinaryPath,
			fork.WithArgs(args...),
		)
		suite.serverProcess = p

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		err := p.Start(ctx)
		if err != nil {
			suite.T().Errorf(
				"Невозможно запустить процесс командой %s: %s. Флаги командной строки: %+v",
				p, err, args,
			)
			return
		}

		err = p.WaitPort(ctx, "tcp", flagServerPort)
		if err != nil {
			suite.T().Errorf("Не удалось дождаться пока порт %s станет доступен для запроса: %s", flagServerPort, err)
			return
		}
	}
}

// TearDownSuite высвобождает имеющиеся зависимости
func (suite *Iteration4Suite) TearDownSuite() {
	suite.stopServer()
}

// TestFlags проверяет, что аргументы командой строки поддерживаются сервером
func (suite *Iteration4Suite) TestFlags() {
	var originalURL, shortenURL string

	errRedirectBlocked := errors.New("HTTP redirect blocked")
	redirPolicy := resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
		return errRedirectBlocked
	})

	restyClient := resty.New()
	transport := restyClient.GetClient().Transport.(*http.Transport)

	resolveIP := "127.0.0.1:" + flagServerPort
	transport.DialContext = mockResolver("tcp", suite.serverAddress, resolveIP)

	httpc := restyClient.
		SetTransport(transport).
		SetBaseURL(suite.serverBaseURL).
		SetRedirectPolicy(redirPolicy)

	suite.Run("shorten", func() {
		originalURL = generateTestURL(suite.T())

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req := httpc.R().
			SetContext(ctx).
			SetHeader("Content-Type", "plain/text").
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
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req := resty.New().
			SetRedirectPolicy(redirPolicy).
			R()
		resp, err := req.
			SetContext(ctx).
			Get(shortenURL)
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
}

// TearDownSuite высвобождает имеющиеся зависимости
func (suite *Iteration4Suite) stopServer() {
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

	// получаем стандартные выводы (логи) процесса
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
