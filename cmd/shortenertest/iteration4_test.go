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

// Iteration4Suite является сьютом с тестами и состоянием для инкремента
type Iteration4Suite struct {
	suite.Suite

	serverAddress string
	serverProcess *fork.BackgroundProcess

	knownEncodingLibs []string
}

// SetupSuite подготавливает необходимые зависимости
func (suite *Iteration4Suite) SetupSuite() {
	// проверяем наличие необходимых флагов
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")
	suite.Require().NotEmpty(flagTargetBinaryPath, "-binary-path non-empty flag required")

	// объявляем список известных библиотек
	suite.knownEncodingLibs = []string{
		"encoding/json",
		"github.com/mailru/easyjson",
		"github.com/pquerna/ffjson",
	}

	suite.serverAddress = "http://localhost:8080"

	// запускаем процесс тестируемого сервера
	{
		envs := os.Environ()
		p := fork.NewBackgroundProcess(context.Background(), flagTargetBinaryPath,
			fork.WithEnv(envs...),
		)
		suite.serverProcess = p

		// ожидаем запуска процесса не более 20 секунд
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		// запускаем процесс
		err := p.Start(ctx)
		if err != nil {
			suite.T().Errorf("Невозможно запустить процесс командой %s: %s. Переменные окружения: %+v", p, err, envs)
			return
		}

		// проверяем, что порт успешно занят процессом
		port := "8080"
		err = p.WaitPort(ctx, "tcp", port)
		if err != nil {
			suite.T().Errorf("Не удалось дождаться пока порт %s станет доступен для запроса: %s", port, err)
			return
		}
	}
}

// TearDownSuite высвобождает имеющиеся зависимости
func (suite *Iteration4Suite) TearDownSuite() {
	// посылаем процессу сигналы для остановки
	exitCode, err := suite.serverProcess.Stop(syscall.SIGINT, syscall.SIGKILL)
	if err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			return
		}
		suite.T().Logf("Не удалось остановить процесс с помощью сигнала ОС: %s", err)
		return
	}

	// проверяем код завешения
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

// TestEncoderUsage пробует рекурсивно найти хотя бы одно использование известных библиотек в директории с исходным кодом проекта
func (suite *Iteration4Suite) TestEncoderUsage() {
	err := usesKnownPackage(suite.T(), flagTargetSourcePath, suite.knownEncodingLibs)
	if err == nil {
		return
	}
	if errors.Is(err, errUsageNotFound) {
		suite.T().Errorf("Не найдено использование известных библиотек кодирования JSON %s", flagTargetSourcePath)
		return
	}
	suite.T().Errorf("Неожиданная ошибка при поиске использования библиотек кодирования JSON по пути %s: %s", flagTargetSourcePath, err)
}

// TestJSONHandler пробует:
// - сгенерировать псевдослучайный URL и передать его в JSON хендлер для сокращения
// - получить оригинальный URL из хендлера редиректа
func (suite *Iteration4Suite) TestJSONHandler() {
	// создаем политику запрещающую редиректы
	errRedirectBlocked := errors.New("HTTP redirect blocked")
	redirPolicy := resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
		return errRedirectBlocked
	})

	// создаем HTTP клиент и подключаем к нему политику редиректов
	httpc := resty.New().
		SetHostURL(suite.serverAddress).
		SetRedirectPolicy(redirPolicy)

	// генерируем URL
	originalURL := generateTestURL(suite.T())
	var shortenURL string

	// пробуем сократить
	suite.Run("shorten", func() {
		// структура тела запроса
		type shortenRequest struct {
			URL string `json:"url"`
		}
		// структура тела ответа
		type shortenResponse struct {
			Result string `json:"result"`
		}

		var result shortenResponse

		// будем ожидать обработки не более 10 секунд
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// подготавливаем и выполняем запрос
		req := httpc.R().
			SetContext(ctx).
			SetHeader("Content-Type", "application/json").
			SetBody(&shortenRequest{
				URL: originalURL,
			}).
			SetResult(&result)
		resp, err := req.Post("/api/shorten")

		noRespErr := suite.Assert().NoError(err, "Ошибка при попытке сделать запрос для сокращения URL")

		shortenURL = result.Result

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
	})

	// пробуем получить оригинальный URL обратно
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
}
