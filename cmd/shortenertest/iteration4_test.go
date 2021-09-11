package main

// Basic imports
import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/suite"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
)

// Iteration4Suite is a suite of autotests
type Iteration4Suite struct {
	suite.Suite

	serverAddress string
	serverProcess *fork.BackgroundProcess

	knownEncodingLibs []string
}

// SetupSuite bootstraps suite dependencies
func (suite *Iteration4Suite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")
	suite.Require().NotEmpty(flagTargetBinaryPath, "-binary-path non-empty flag required")

	suite.knownEncodingLibs = []string{
		"encoding/json",
		"github.com/mailru/easyjson",
		"github.com/pquerna/ffjson",
	}

	suite.serverAddress = "http://localhost:8080"

	// start server
	{
		envs := os.Environ()
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
func (suite *Iteration4Suite) TearDownSuite() {
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

// TestEncoderUsage attempts to recursively find usage of known HTTP frameworks in given sources
func (suite *Iteration4Suite) TestEncoderUsage() {
	err := filepath.WalkDir(flagTargetSourcePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			// skip vendor directory
			if d.Name() == "vendor" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			// dive into regular directory
			return nil
		}

		// skip test files or non-Go files
		if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}

		spec, err := importsKnownPackage(suite.T(), path, suite.knownEncodingLibs)
		if err != nil {
			// log error and continue traversing
			suite.T().Logf("Ошибка инспекции файла %s: %s", path, err)
			return nil
		}
		if spec != nil && spec.Name.String() != "_" {
			return errUsageFound
		}

		return nil
	})

	if errors.Is(err, errUsageFound) {
		return
	}

	if err == nil {
		suite.T().Errorf("Не найдено использование известных библиотек кодирования JSON %s", flagTargetSourcePath)
		return
	}
	suite.T().Errorf("Неожиданная ошибка при поиске использования библиотек кодирования JSON по пути %s: %s", flagTargetSourcePath, err)
}

// TestJSONHandler attempts to:
// - generate and send random URL to JSON API handler
// - fetch original URL by sending shorten URL to expand handler
func (suite *Iteration4Suite) TestJSONHandler() {
	// create HTTP client without redirects support
	errRedirectBlocked := errors.New("HTTP redirect blocked")
	redirPolicy := resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
		return errRedirectBlocked
	})

	httpc := resty.New().
		SetHostURL(suite.serverAddress).
		SetRedirectPolicy(redirPolicy)

	// declare and generate URLs
	originalURL := generateTestURL(suite.T())
	var shortenURL string

	suite.Run("shorten", func() {
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
		if !errors.Is(err, errRedirectBlocked) {
			dump := dumpRequest(req.RawRequest, true)
			suite.Require().NoErrorf(err, "Ошибка при попытке сделать запрос для сокращения URL:\n\n %s", dump)
		}

		shortenURL = result.Result

		suite.Assert().Containsf(resp.Header().Get("Content-Type"), "application/json",
			"Заголовок ответа Content-Type содержит несоответствующее значение",
		)
		suite.Assert().Equalf(http.StatusCreated, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL)
		suite.Assert().NoErrorf(func() error {
			_, err := url.Parse(shortenURL)
			return err
		}(), "Невозможно распарсить полученный сокращенный URL - %s : %s", shortenURL, err)
	})

	suite.Run("expand", func() {
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
	})
}
