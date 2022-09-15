package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/stdlib"
	"github.com/stretchr/testify/suite"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
)

// Iteration12Suite является сьютом с тестами и состоянием для инкремента
type Iteration12Suite struct {
	suite.Suite

	serverAddress string
	serverProcess *fork.BackgroundProcess

	dbconn *sql.DB
}

// SetupSuite подготавливает необходимые зависимости
func (suite *Iteration12Suite) SetupSuite() {
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
func (suite *Iteration12Suite) TearDownSuite() {
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

// TestBatchShorten attempts to:
// - generate and send random URLs to batch shorten handler
// - expand fetched short urls and match them with original ones
func (suite *Iteration12Suite) TestBatchShorten() {
	type shortenRequest struct {
		CorrelationID string `json:"correlation_id"`
		OriginalURL   string `json:"original_url"`
	}

	type shortenResponse struct {
		CorrelationID string `json:"correlation_id"`
		ShortURL      string `json:"short_url"`
	}

	var responseData []shortenResponse
	requestData := []shortenRequest{
		{
			CorrelationID: uuid.Must(uuid.NewV4()).String(),
			OriginalURL:   generateTestURL(suite.T()),
		},
		{
			CorrelationID: uuid.Must(uuid.NewV4()).String(),
			OriginalURL:   generateTestURL(suite.T()),
		},
	}

	// correlations between originalURLs and shortURLs
	correlations := make(map[string]string)

	// create HTTP client without redirects support
	errRedirectBlocked := errors.New("HTTP redirect blocked")
	redirPolicy := resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
		return errRedirectBlocked
	})

	httpc := resty.New().
		SetHostURL(suite.serverAddress).
		SetRedirectPolicy(redirPolicy)

	suite.Run("shorten_batch", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req := httpc.R().
			SetContext(ctx).
			SetHeader("Content-Type", "application/json").
			SetBody(requestData).
			SetResult(&responseData)
		resp, err := req.Post("/api/shorten/batch")

		noRespErr := suite.Assert().NoError(err, "Ошибка при попытке сделать запрос для множественного сокращения URL")

		validStatus := suite.Assert().Equalf(http.StatusCreated, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)

		validContentType := suite.Assert().Containsf(resp.Header().Get("Content-Type"), "application/json",
			"Заголовок ответа Content-Type содержит несоответствующее значение",
		)

		suite.Assert().Len(responseData, len(requestData), "Кол-во объектов в ответе не совпадает с кол-вом объектов в запросе")

		allCorrelationsFound := true
		for _, respPair := range responseData {
			var originalURL string
			for _, reqPair := range requestData {
				if respPair.CorrelationID == reqPair.CorrelationID {
					originalURL = reqPair.OriginalURL
					break
				}
			}

			found := suite.Assert().NotEmptyf(originalURL, "Не удалось найти оригинальный URL по correlation ID: %s", respPair.CorrelationID)
			if !found {
				allCorrelationsFound = false
			}

			correlations[respPair.ShortURL] = originalURL
		}

		if !noRespErr || !validStatus || !validContentType || !allCorrelationsFound {
			dump := dumpRequest(req.RawRequest, true)
			jsonBody, _ := json.Marshal(requestData)
			suite.T().Logf("Оригинальный запрос:\n\n%s\n\nТело запроса:\n\n%s", dump, jsonBody)
		}
	})

	suite.Run("expand", func() {
		for shortenURL, originalURL := range correlations {
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
}
