package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/stdlib"
	"github.com/stretchr/testify/suite"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
)

// Iteration13Suite является сьютом с тестами и состоянием для инкремента
type Iteration13Suite struct {
	suite.Suite

	serverAddress string
	serverProcess *fork.BackgroundProcess

	dbconn *sql.DB
}

// SetupSuite подготавливает необходимые зависимости
func (suite *Iteration13Suite) SetupSuite() {
	// проверяем наличие необходимых флагов
	suite.Require().NotEmpty(flagTargetBinaryPath, "-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagDatabaseDSN, "-database-dsn non-empty flag required")

	suite.serverAddress = "http://localhost:8080"

	// запускаем процесс тестируемого сервера
	{
		envs := os.Environ()
		args := []string{"-d=" + flagDatabaseDSN}
		p := fork.NewBackgroundProcess(context.Background(), flagTargetBinaryPath,
			fork.WithEnv(envs...),
			fork.WithArgs(args...),
		)
		suite.serverProcess = p

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		err := p.Start(ctx)
		if err != nil {
			suite.T().Errorf("Невозможно запустить процесс командой %s: %s. Переменные окружения: %+v, аргументы: %+v", p, err, envs, args)
			return
		}

		port := "8080"
		err = p.WaitPort(ctx, "tcp", port)
		if err != nil {
			suite.T().Errorf("Не удалось дождаться пока порт %s станет доступен для запроса: %s", port, err)
			return
		}
	}

	// connect to database
	{
		// disable prepared statements
		driverConfig := stdlib.DriverConfig{
			ConnConfig: pgx.ConnConfig{
				PreferSimpleProtocol: true,
			},
		}
		stdlib.RegisterDriverConfig(&driverConfig)

		conn, err := sql.Open("pgx", driverConfig.ConnectionString(flagDatabaseDSN))
		if err != nil {
			suite.T().Errorf("Не удалось подключиться к базе данных: %s", err)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err = conn.PingContext(ctx); err != nil {
			suite.T().Errorf("Не удалось подключиться проверить подключение к базе данных: %s", err)
			return
		}

		suite.dbconn = conn
	}
}

// TearDownSuite высвобождает имеющиеся зависимости
func (suite *Iteration13Suite) TearDownSuite() {
	if suite.dbconn != nil {
		_ = suite.dbconn.Close()
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

// TestConflict attempts to:
// - generate and send random URL to shorten handler multiple times
// - expect to get 409 status on duplicate attempts
func (suite *Iteration13Suite) TestConflict() {
	jar, err := cookiejar.New(nil)
	suite.Require().NoError(err, "Неожиданная ошибка при создании Cookie Jar")

	httpc := resty.New().
		SetHostURL(suite.serverAddress).
		SetHeader("Accept-Encoding", "identity").
		SetCookieJar(jar)

	suite.Run("shorten", func() {
		originalURL := generateTestURL(suite.T())
		var shortenURL string

		for attempt := 0; attempt <= 2; attempt++ {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			req := httpc.R().
				SetContext(ctx).
				SetBody(originalURL)
			resp, err := req.Post("/")

			noRespErr := suite.Assert().NoError(err, "Ошибка при попытке сделать запрос для сокращения URL")

			// save original shorten URL
			if shortenURL == "" {
				shortenURL = string(resp.Body())
			} else {
				conflictURL := string(resp.Body())
				suite.Assert().Equal(shortenURL, conflictURL, "Несовпадение сокращенных URL при конфликте")
			}

			expectedStatus := http.StatusCreated
			if attempt > 0 {
				expectedStatus = http.StatusConflict
			}

			validStatus := suite.Assert().Equalf(expectedStatus, resp.StatusCode(),
				"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

			_, urlParseErr := url.Parse(shortenURL)
			validURL := suite.Assert().NoErrorf(urlParseErr,
				"Невозможно распарсить полученный сокращенный URL - %s : %s", shortenURL, err,
			)

			if !noRespErr || !validStatus || !validURL {
				dump := dumpRequest(req.RawRequest, true)
				suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
			}
		}
	})

	suite.Run("shorten_api", func() {
		type shortenRequest struct {
			URL string `json:"url"`
		}

		type shortenResponse struct {
			Result string `json:"result"`
		}

		originalURL := generateTestURL(suite.T())
		var shortenURL string

		for attempt := 0; attempt <= 2; attempt++ {
			var result shortenResponse

			req := httpc.R().
				SetHeader("Content-Type", "application/json").
				SetBody(&shortenRequest{
					URL: originalURL,
				}).
				SetError(&result)
			resp, err := req.Post("/api/shorten")

			noRespErr := suite.Assert().NoError(err, "Ошибка при попытке сделать запрос для сокращения URL")

			// save original shorten URL
			if shortenURL == "" {
				shortenURL = result.Result
			} else {
				conflictURL := result.Result
				suite.Assert().Equal(shortenURL, conflictURL, "Несовпадение сокращенных URL при конфликте")
			}

			expectedStatus := http.StatusCreated
			if attempt > 0 {
				expectedStatus = http.StatusConflict
			}

			validStatus := suite.Assert().Equalf(expectedStatus, resp.StatusCode(),
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
		}
	})
}
