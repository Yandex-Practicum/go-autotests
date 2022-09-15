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
type Iteration7Suite struct {
	suite.Suite

	serverAddress    string
	serverBaseURL    string
	serverProcess    *fork.BackgroundProcess
	knownPgLibraries []string
}

// SetupSuite подготавливает необходимые зависимости
func (suite *Iteration7Suite) SetupSuite() {
	// проверяем наличие необходимых флагов
	suite.Require().NotEmpty(flagTargetBinaryPath, "-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagServerPort, "-server-port non-empty flag required")
	suite.Require().NotEmpty(flagFileStoragePath, "-file-storage-path non-empty flag required")
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")

	// список известных драйверов для PostgreSQL
	suite.knownPgLibraries = []string{
		"database/sql",
		"github.com/jackc/pgx",
		"github.com/lib/pq",
	}

	// запускаем процесс тестируемого сервера
	{
		suite.serverAddress = "localhost:" + flagServerPort
		suite.serverBaseURL = "http://" + suite.serverAddress

		envs := append(os.Environ(), []string{
			"SERVER_ADDRESS=" + suite.serverAddress,
		}...)
		// передаем флаги в процесс сервера
		args := []string{
			"-b=" + suite.serverBaseURL,
			"-f=" + flagFileStoragePath,
		}

		p := fork.NewBackgroundProcess(context.Background(), flagTargetBinaryPath,
			fork.WithEnv(envs...),
			fork.WithArgs(args...),
		)
		suite.serverProcess = p

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		err := p.Start(ctx)
		if err != nil {
			suite.T().Errorf(
				"Невозможно запустить процесс командой %s: %s. Переменные окружения: %+v, флаги командной строки: %+v",
				p, err, envs, args,
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
func (suite *Iteration7Suite) TearDownSuite() {
	suite.stopServer()
}

// TestFlags повторяет тесты предыдущих итераций и проверяет, что аргументы командой строки поддерживаются сервером
func (suite *Iteration7Suite) TestFlags() {
	shortenURLs := make(map[string]string)

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
		SetHostURL(suite.serverBaseURL).
		SetRedirectPolicy(redirPolicy)

	suite.Run("shorten", func() {
		originalURL := generateTestURL(suite.T())

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req := httpc.R().
			SetContext(ctx).
			SetBody(originalURL)
		resp, err := req.Post("/")

		noRespErr := suite.Assert().NoError(err, "Ошибка при попытке сделать запрос для сокращения URL")

		shortenURL := string(resp.Body())

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

		shortenURLs[originalURL] = shortenURL
	})

	suite.Run("shorten_api", func() {
		originalURL := generateTestURL(suite.T())

		type shortenRequest struct {
			URL string `json:"url"`
		}

		type shortenResponse struct {
			Result string `json:"result"`
		}

		var result shortenResponse

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req := httpc.R().
			SetContext(ctx).
			SetHeader("Content-Type", "application/json").
			SetBody(&shortenRequest{
				URL: originalURL,
			}).
			SetResult(&result)
		resp, err := req.Post("/api/shorten")

		noRespErr := suite.Assert().NoError(err, "Ошибка при попытке сделать запрос для сокращения URL")

		shortenURL := result.Result

		validStatus := suite.Assert().Equalf(http.StatusCreated, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

		validContentType := suite.Assert().Containsf(resp.Header().Get("Content-Type"), "application/json",
			"Заголовок ответа Content-Type содержит несоответствующее значение",
		)

		_, urlParseErr := url.Parse(shortenURL)
		validURL := suite.Assert().NoErrorf(urlParseErr,
			"Невозможно распарсить полученный сокращенный URL - %s : %s", shortenURL, err,
		)

		if !noRespErr || !validStatus || !validContentType || !validURL {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}

		shortenURLs[originalURL] = shortenURL
	})

	suite.Run("expand", func() {
		for originalURL, shortenURL := range shortenURLs {
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
		}
	})

	suite.Run("check_file", func() {
		err := usesKnownPackage(suite.T(), flagTargetSourcePath, suite.knownPgLibraries)
		if err == nil {
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

// TearDownSuite высвобождает имеющиеся зависимости
func (suite *Iteration7Suite) stopServer() {
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
