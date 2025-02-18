package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/suite"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
)

// AccrualSuite is a suite of autotests
type AccrualSuite struct {
	suite.Suite

	accrualServerAddress string
	accrualProcess       *fork.BackgroundProcess
}

// SetupSuite bootstraps suite dependencies
func (suite *AccrualSuite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagAccrualBinaryPath, "-accrual-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagAccrualDatabaseURI, "-accrual-database-uri non-empty flag required")
	suite.Require().NotEmpty(flagAccrualHost, "-accrual-host non-empty flag required")
	suite.Require().NotEmpty(flagAccrualPort, "-accrual-port non-empty flag required")

	// start accrual server
	{
		suite.accrualServerAddress = "http://" + flagAccrualHost + ":" + flagAccrualPort

		envs := append(os.Environ(),
			"RUN_ADDRESS="+flagAccrualHost+":"+flagAccrualPort,
			"DATABASE_URI="+flagAccrualDatabaseURI,
		)
		p := fork.NewBackgroundProcess(context.Background(), flagAccrualBinaryPath,
			fork.WithEnv(envs...),
		)

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		err := p.Start(ctx)
		if err != nil {
			suite.T().Errorf("Невозможно запустить процесс командой %s: %s. Переменные окружения: %+v", p, err, envs)
			return
		}

		suite.accrualProcess = p

		port := flagAccrualPort
		err = p.WaitPort(ctx, "tcp", port)
		if err != nil {
			suite.T().Errorf("Не удалось дождаться пока порт %s станет доступен для запроса: %s", port, err)
			return
		}
	}
}

// TearDownSuite teardowns suite dependencies
func (suite *AccrualSuite) TearDownSuite() {
	if suite.accrualProcess == nil {
		return
	}

	exitCode, err := suite.accrualProcess.Stop(syscall.SIGINT, syscall.SIGKILL)
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

	// try to read stdout/stderr
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	out := suite.accrualProcess.Stderr(ctx)
	if len(out) > 0 {
		suite.T().Logf("Получен STDERR лог процесса:\n\n%s", string(out))
	}
	out = suite.accrualProcess.Stdout(ctx)
	if len(out) > 0 {
		suite.T().Logf("Получен STDOUT лог процесса:\n\n%s", string(out))
	}
}

// TestRegisterMechanic checks orderAccrual mechanics register handler
func (suite *AccrualSuite) TestRegisterMechanic() {
	httpc := resty.New().
		SetHostURL(suite.accrualServerAddress)

	suite.Run("bad_match", func() {
		m := []byte(`
			{
				"match": "",
				"reward": 10,
				"reward_type": "%"
			}
		`)

		req := httpc.R().
			SetHeader("Content-Type", "application/json").
			SetBody(m)

		resp, err := req.Post("/api/goods")

		noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на регистрацию механики")
		validStatus := suite.Assert().Equalf(http.StatusBadRequest, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)

		if !noRespErr || !validStatus {
			dump := dumpRequest(suite.T(), req.RawRequest, bytes.NewReader(m))
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("bad_reward", func() {
		m := []byte(`
			{
				"match": "Milka",
				"reward": -10,
				"reward_type": "%"
			}
		`)

		req := httpc.R().
			SetHeader("Content-Type", "application/json").
			SetBody(m)

		resp, err := req.Post("/api/goods")

		noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на регистрацию механики")
		validStatus := suite.Assert().Equalf(http.StatusBadRequest, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)

		if !noRespErr || !validStatus {
			dump := dumpRequest(suite.T(), req.RawRequest, bytes.NewReader(m))
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("bad_reward_type", func() {
		m := []byte(`
			{
				"match": "Milka",
				"reward": 10,
				"reward_type": "USD"
			}
		`)

		req := httpc.R().
			SetHeader("Content-Type", "application/json").
			SetBody(m)

		resp, err := req.Post("/api/goods")

		noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на регистрацию механики")
		validStatus := suite.Assert().Equalf(http.StatusBadRequest, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)

		if !noRespErr || !validStatus {
			dump := dumpRequest(suite.T(), req.RawRequest, bytes.NewReader(m))
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("successful_register", func() {
		m := []byte(`
			{
				"match": "Milka",
				"reward": 11.5,
				"reward_type": "%"
			}
		`)

		req := httpc.R().
			SetHeader("Content-Type", "application/json").
			SetBody(m)

		resp, err := req.Post("/api/goods")

		noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на регистрацию механики")
		validStatus := suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)

		if !noRespErr || !validStatus {
			dump := dumpRequest(suite.T(), req.RawRequest, bytes.NewReader(m))
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("duplicate_mechanic", func() {
		m := []byte(`
			{
				"match": "Milka",
				"reward": 42,
				"reward_type": "pt"
			}
		`)

		req := httpc.R().
			SetHeader("Content-Type", "application/json").
			SetBody(m)

		resp, err := req.Post("/api/goods")

		noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на регистрацию механики")
		validStatus := suite.Assert().Equalf(http.StatusConflict, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)

		if !noRespErr || !validStatus {
			dump := dumpRequest(suite.T(), req.RawRequest, bytes.NewReader(m))
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})
}

// TestRegisterOrder checks orderAccrual order register handler
func (suite *AccrualSuite) TestRegisterOrder() {
	httpc := resty.New().
		SetHostURL(suite.accrualServerAddress)

	successOrderNumber, err := generateOrderNumber(suite.T())
	suite.Require().NoErrorf(err, "не удалось сгенерировать номер заказа")

	suite.Run("bad_order_number", func() {
		o := []byte(`
			{
				"order": "12345678901",
				"goods": [
					{
						"description": "Стиральная машинка LG",
						"price": 47399.99
					}
				]
			}
		`)

		req := httpc.R().
			SetHeader("Content-Type", "application/json").
			SetBody(o)

		resp, err := req.Post("/api/orders")

		noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на регистрацию механики")
		validStatus := suite.Assert().Equalf(http.StatusBadRequest, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)

		if !noRespErr || !validStatus {
			dump := dumpRequest(suite.T(), req.RawRequest, bytes.NewReader(o))
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("successful_registration", func() {
		o := []byte(`
			{
				"order": "` + successOrderNumber + `",
				"goods": [
					{
						"description": "Стиральная машинка LG",
						"price": 47399.99
					}
				]
			}
		`)

		req := httpc.R().
			SetHeader("Content-Type", "application/json").
			SetBody(o)

		resp, err := req.Post("/api/orders")

		noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на регистрацию механики")
		validStatus := suite.Assert().Equalf(http.StatusAccepted, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)

		if !noRespErr || !validStatus {
			dump := dumpRequest(suite.T(), req.RawRequest, bytes.NewReader(o))
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("duplicate_order", func() {
		o := []byte(`
			{
				"order": "` + successOrderNumber + `",
				"goods": [
					{
						"description": "Холодильник Beko",
						"price": 22599.99
					}
				]
			}
		`)

		req := httpc.R().
			SetHeader("Content-Type", "application/json").
			SetBody(o)

		resp, err := req.Post("/api/orders")

		noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на регистрацию механики")
		validStatus := suite.Assert().Equalf(http.StatusConflict, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)

		if !noRespErr || !validStatus {
			dump := dumpRequest(suite.T(), req.RawRequest, bytes.NewReader(o))
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})
}

// TestEndToEnd attempts to:
// - register new mechanics
// - register new order with appropriate goods
// - check orderAccrual amount
func (suite *AccrualSuite) TestEndToEnd() {
	httpc := resty.New().
		SetHostURL(suite.accrualServerAddress)

	orderNumber, err := generateOrderNumber(suite.T())
	suite.Require().NoError(err, "не удалось сгенерировать номер заказа")

	suite.Run("register_mechanics", func() {
		mechanics := [][]byte{
			[]byte(`{"match": "Pringles", "reward": 10, "reward_type": "%"}`),
			[]byte(`{"match": "Coca-Cola", "reward": 12.5, "reward_type": "pt"}`),
		}

		for _, mechanic := range mechanics {
			req := httpc.R().
				SetHeader("Content-Type", "application/json").
				SetBody(mechanic)

			resp, err := req.Post("/api/goods")

			noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на регистрацию механики")
			validStatus := suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
				"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
			)

			if !noRespErr || !validStatus {
				dump := dumpRequest(suite.T(), req.RawRequest, bytes.NewReader(mechanic))
				suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
			}
		}
	})

	suite.Run("register_order", func() {
		order := []byte(`{
			"order": "` + orderNumber + `",
			"goods": [
				{
					"description": "Чайник Tefal",
					"price": 7000
				},
				{
					"description": "Чипсы Pringles бекон",
					"price": 147.5
				},
				{
					"description": "Напиток Coca-Cola Zero 0.5л",
					"price": 55
				}
			]
		}`)

		req := httpc.R().
			SetHeader("Content-Type", "application/json").
			SetBody(order)

		resp, err := req.Post("/api/orders")

		noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на регистрацию заказа")
		validStatus := suite.Assert().Equalf(http.StatusAccepted, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)

		if !noRespErr || !validStatus {
			dump := dumpRequest(suite.T(), req.RawRequest, bytes.NewReader(order))
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("fetch_accrual", func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		for {
			select {
			case <-ctx.Done():
				suite.T().Errorf("не удалось дождаться окончания расчета за 10 секунд")
				return
			case <-ticker.C:
				var acc orderAccrual

				req := httpc.R().
					SetResult(&acc).
					SetPathParam("number", orderNumber)

				resp, err := req.Get("/api/orders/{number}")

				noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на регистрацию заказа")
				validStatus := suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
					"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
				)

				expectedStatus := suite.Assert().NotEqualf("INVALID", acc.Status, "неожиданный статус расчета начисления")
				if acc.Status != "PROCESSED" {
					continue
				}

				expectedAccrual := suite.Assert().Equalf(float32(27.25), acc.Accrual, "не верная сумма начислений за заказ")

				// success
				if noRespErr && validStatus && expectedStatus && expectedAccrual {
					return
				}

				dump := dumpRequest(suite.T(), req.RawRequest, nil)
				suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
			}
		}
	})
}
