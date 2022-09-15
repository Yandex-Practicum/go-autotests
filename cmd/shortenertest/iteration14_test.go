package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/stdlib"
	"github.com/stretchr/testify/suite"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
)

// Iteration14Suite является сьютом с тестами и состоянием для инкремента
type Iteration14Suite struct {
	suite.Suite

	serverAddress string
	serverProcess *fork.BackgroundProcess

	dbconn *sql.DB
}

// SetupSuite подготавливает необходимые зависимости
func (suite *Iteration14Suite) SetupSuite() {
	// проверяем наличие необходимых флагов
	suite.Require().NotEmpty(flagTargetBinaryPath, "-binary-path non-empty flag required")
	if flagDatabaseDSN == "" {
		suite.T().Log("WARN: -database-dsn flag not provided, will run degradation tests")
	}

	suite.serverAddress = "http://localhost:8080"

	// запускаем процесс тестируемого сервера
	{
		envs := os.Environ()

		var args []string
		if flagDatabaseDSN != "" {
			args = append(args, "-d="+flagDatabaseDSN)
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
	if flagDatabaseDSN != "" {
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
func (suite *Iteration14Suite) TearDownSuite() {
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

// TestDelete attempts to:
// - generate and send random URLs to shorten handler
// - send DELETE request with given URLs
// - check URL status afterwards
func (suite *Iteration14Suite) TestDelete() {
	jar, err := cookiejar.New(nil)
	suite.Require().NoError(err, "Неожиданная ошибка при создании Cookie Jar")

	// create HTTP client without redirects support
	errRedirectBlocked := errors.New("HTTP redirect blocked")
	redirPolicy := resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
		return errRedirectBlocked
	})

	httpc := resty.New().
		SetHostURL(suite.serverAddress).
		SetHeader("Accept-Encoding", "identity").
		SetCookieJar(jar).
		SetRedirectPolicy(redirPolicy)

	shortenURLs := make(map[string]string)

	suite.Run("shorten", func() {
		for num := 0; num <= 10; num++ {
			originalURL := generateTestURL(suite.T())

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			req := httpc.R().
				SetContext(ctx).
				SetBody(originalURL)
			resp, err := req.Post("/")

			noRespErr := suite.Assert().NoError(err, "Ошибка при попытке сделать запрос для сокращения URL")

			shortenURL := string(resp.Body())
			shortenURLs[originalURL] = shortenURL

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
		}
	})

	suite.Run("remove", func() {
		var body []string
		for _, shorten := range shortenURLs {
			u, err := url.Parse(shorten)
			if err != nil {
				continue
			}
			body = append(body, strings.Trim(u.Path, "/"))
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req := httpc.R().
			SetContext(ctx).
			SetHeader("Content-Type", "application/json").
			SetBody(body)
		resp, err := req.Delete("/api/user/urls")

		noRespErr := suite.Assert().NoError(err, "Ошибка при попытке сделать запрос для удаления URL")

		validStatus := suite.Assert().Equalf(http.StatusAccepted, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

		if !noRespErr || !validStatus {
			dump := dumpRequest(req.RawRequest, true)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("check_state", func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				suite.T().Errorf("Не удалось дождаться удаления переданных URL в течении 60 секунд")
				return
			case <-ticker.C:
				var deletedCount int
				for _, shorten := range shortenURLs {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second)
					defer cancel()

					resp, err := httpc.R().
						SetContext(ctx).
						Get(shorten)
					if err == nil && resp != nil && resp.StatusCode() == http.StatusGone {
						deletedCount++
					}
				}
				if deletedCount == len(shortenURLs) {
					return
				}
			}
		}
	})
}

// TestDeleteConcurrent is a concurrent version of TestDelete
func (suite *Iteration14Suite) TestDeleteConcurrent() {
	jar, err := cookiejar.New(nil)
	suite.Require().NoError(err, "Неожиданная ошибка при создании Cookie Jar")

	// create HTTP client without redirects support
	errRedirectBlocked := errors.New("HTTP redirect blocked")
	redirPolicy := resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
		return errRedirectBlocked
	})

	httpc := resty.New().
		SetHostURL(suite.serverAddress).
		SetHeader("Accept-Encoding", "identity").
		SetCookieJar(jar).
		SetRedirectPolicy(redirPolicy)

	// urlProcessor runs single URL processing sequence
	urlProcessor := func() {
		var shortenURL string

		// shorten URL
		{
			originalURL := generateTestURL(suite.T())

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			req := httpc.R().
				SetContext(ctx).
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
		}

		// remove URL
		{
			u, err := url.Parse(shortenURL)
			validURL := suite.Assert().NoErrorf(err,
				"Невозможно распарсить полученный сокращенный URL - %s : %s", shortenURL, err,
			)

			body := []string{strings.Trim(u.Path, "/")}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			req := httpc.R().
				SetContext(ctx).
				SetHeader("Content-Type", "application/json").
				SetBody(body)
			resp, err := req.Delete("/api/user/urls")

			noRespErr := suite.Assert().NoError(err, "Ошибка при попытке сделать запрос для удаления URL")

			validStatus := suite.Assert().Equalf(http.StatusAccepted, resp.StatusCode(),
				"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

			if !noRespErr || !validStatus || !validURL {
				dump := dumpRequest(req.RawRequest, true)
				suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
			}
		}

		// check URL state
		{
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()

			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			func() {
				for {
					select {
					case <-ctx.Done():
						suite.T().Errorf("Не удалось дождаться удаления переданных URL в течении 60 секунд")
						return
					case <-ticker.C:
						ctx, cancel := context.WithTimeout(context.Background(), time.Second)
						defer cancel()

						resp, err := httpc.R().
							SetContext(ctx).
							Get(shortenURL)
						if err == nil && resp != nil && resp.StatusCode() == http.StatusGone {
							return
						}
					}
				}
			}()
		}
	}

	// run multiple requests sequences at once
	for i := 0; i <= 20; i++ {
		urlProcessor()
	}
}
