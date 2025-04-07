package main

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
)

type Sprint6FinalSuite struct {
	suite.Suite

	serverProcess *fork.BackgroundProcess
	serverAddress string
}

func (suite *Sprint6FinalSuite) SetupSuite() {
	suite.Require().NotEmpty(flagServerBinaryPath, "-server-binary-path non-empty flag required")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	suite.serverUp(ctx)
}

func (suite *Sprint6FinalSuite) serverUp(ctx context.Context) {
	suite.serverProcess = fork.NewBackgroundProcess(context.Background(), flagServerBinaryPath)

	err := suite.serverProcess.Start(ctx)
	if err != nil {
		suite.T().Errorf("Невозможно запустить процесс командой %q: %v", suite.serverProcess, err)
		return
	}

	const port = ":8080"
	err = suite.serverProcess.WaitPort(ctx, "tcp", port)
	if err != nil {
		suite.T().Errorf("Не удалось дождаться пока порт %s станет доступен для запроса: %v", port, err)
		return
	}

	suite.serverAddress = "http://localhost" + port
}

func (suite *Sprint6FinalSuite) TearDownSuite() {
	suite.serverShutdown()
}

func (suite *Sprint6FinalSuite) serverShutdown() {
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
		suite.T().Logf("Процесс завершился с ненулевым статусом %d", exitCode)
	}

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

func (suite *Sprint6FinalSuite) TestSprint6Final() {
	httpc := resty.New().
		SetBaseURL(suite.serverAddress)

	suite.Run("serve_index", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		resp, err := httpc.R().
			SetContext(ctx).
			Get("/")

		suite.Require().NoError(err, "Ошибка при попытке получить HTML страницу")
		suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере GET /")
		suite.Assert().Containsf(resp.Header().Get("Content-Type"), "text/html",
			"Заголовок ответа Content-Type не содержит ожидаемое значение")
	})

	suite.Run("upload_text_to_morse", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		resp := sendMorseCodeRequest(suite.T(), httpc, ctx, "test12")

		suite.Assert().Containsf(strings.TrimSpace(resp), ".--. .-. .. .-- . -", "Ответ не содержит ожидаемый код Морзе")
	})

	suite.Run("upload_morse_to_text", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		resp := sendMorseCodeRequest(suite.T(), httpc, ctx, "test")

		suite.Assert().Containsf(strings.TrimSpace(resp), "ПРИВЕТ", "Ответ не содержит ожидаемый текст")
	})

	suite.Run("upload_random", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		tmpDir, err := os.MkdirTemp("", "test-dir-*")
		suite.Require().NoError(err, "Ошибка при создании временной директории")
		//defer os.RemoveAll(tmpDir)

		alphabeth := []rune("АБВГДЕЖЗИЙКЛМНОПРСТУФХЦЧШЩЫЭЮЯ")
		length := rand.Intn(20) + 10
		text := make([]rune, length)
		for i := range text {
			text[i] = alphabeth[rand.Intn(len(alphabeth))]
		}
		originalText := string(text)

		suite.T().Logf("originalText: %s", originalText)

		textFilePath := filepath.Join(tmpDir, "random.txt")
		err = os.WriteFile(textFilePath, []byte(originalText), 0644)
		suite.Require().NoError(err, "Ошибка при записи в файл с исходным текстом")

		resp := sendMorseCodeRequest(suite.T(), httpc, ctx, textFilePath)
		suite.Assert().Containsf(resp, ".-", "Ответ должен содержать код Морзе")

		morseFilePath := filepath.Join(tmpDir, "morse.txt")
		err = os.WriteFile(morseFilePath, []byte(resp), 0644)
		suite.Require().NoError(err, "Ошибка при записи кода Морзе в файл")

		resp = sendMorseCodeRequest(suite.T(), httpc, ctx, morseFilePath)
		suite.Assert().Containsf(resp, originalText, "Ответ должен содержать исходный текст")
	})
}

func sendMorseCodeRequest(t *testing.T, httpc *resty.Client, ctx context.Context, inFile string) (response string) {
	t.Helper()

	file, err := os.Open(inFile)
	require.NoError(t, err, "Не удалось открыть файл %s", inFile)
	defer file.Close()

	resp, err := httpc.R().
		SetContext(ctx).
		SetFileReader("myFile", inFile, file).
		Post("/upload")

	require.NoError(t, err, "Ошибка при попытке загрузить файл")
	assert.Equalf(t, http.StatusOK, resp.StatusCode(), "Несоответствие статус кода ответа ожидаемому в хендлере POST /upload")

	return string(resp.Body())
}
