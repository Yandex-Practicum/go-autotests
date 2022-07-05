package main

import (
	"bytes"
	"context"
	mathrand "math/rand"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/Yandex-Practicum/go-autotests/internal/random"
)

// TestEndToEnd does the following:
// - registers new mechanics in accrual service
// - creates new order and sends it to accrual
// - registers new user
// - uploads user's order number
// - waits for accrual to be performed
// - checks balance
// - performs partial balance withdrawal
func (suite *GophermartSuite) TestEndToEnd() {
	trademark := random.ASCIIString(10, 20)
	expectedAccrual := float32(729.98)

	orderNum, err := generateOrderNumber(suite.T())
	suite.Require().NoError(err, "Не удалось сгенерировать номер заказа")

	suite.Run("register_accrual_mechanic", func() {
		m := []byte(`
			{
				"match": "` + trademark + `",
				"reward": 5,
				"reward_type": "%"
			}
		`)

		req := resty.New().
			SetHostURL(suite.accrualServerAddress).
			R().
			SetHeader("Content-Type", "application/json").
			SetBody(m)

		resp, err := req.Post("/api/goods")

		noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на регистрацию механики в системе расчета баллов лояльности")
		validStatus := suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)

		if !noRespErr || !validStatus {
			dump := dumpRequest(suite.T(), req.RawRequest, bytes.NewReader(m))
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("register_order_for_accrual", func() {
		o := []byte(`
			{
				"order": "` + orderNum + `",
				"goods": [
					{
						"description": "Стиральная машинка LG",
						"price": 47399.99
					},
					{
						"description": "Телевизор ` + trademark + `",
						"price": 14599.50
					}
				]
			}
		`)

		req := resty.New().
			SetHostURL(suite.accrualServerAddress).
			R().
			SetHeader("Content-Type", "application/json").
			SetBody(o)

		resp, err := req.Post("/api/orders")

		noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на регистрацию заказа в системе расчета баллов лояльности")
		validStatus := suite.Assert().Equalf(http.StatusAccepted, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)

		if !noRespErr || !validStatus {
			dump := dumpRequest(suite.T(), req.RawRequest, bytes.NewReader(o))
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	jar, err := cookiejar.New(nil)
	suite.Require().NoError(err, "Не удалось создать объект cookie jar")

	httpc := resty.New().
		SetHostURL(suite.gophermartServerAddress).
		SetCookieJar(jar)

	suite.Run("register_user", func() {
		login := random.ASCIIString(7, 14)
		password := random.ASCIIString(16, 32)

		m := []byte(`{"login": "` + login + `","password": "` + password + `"}`)

		req := httpc.R().
			SetHeader("Content-Type", "application/json").
			SetBody(m)

		resp, err := req.Post("/api/user/register")

		noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на регистрацию пользователя в системе лояльности")
		validStatus := suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)

		authHeader := resp.Header().Get("Authorization")
		setCookieHeader := resp.Header().Get("Set-Cookie")
		hasAuthorization := suite.Assert().True(authHeader != "" || setCookieHeader != "",
			"Не удалось обнаружить авторизационные данные в ответе")

		if !noRespErr || !validStatus || !hasAuthorization {
			dump := dumpRequest(suite.T(), req.RawRequest, bytes.NewReader(m))
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}

		// store auth header
		if authHeader != "" {
			httpc.SetHeader("Authorization", authHeader)
		}
	})

	suite.Run("order_upload", func() {
		body := []byte(orderNum)

		req := httpc.R().
			SetHeader("Content-Type", "text/plain").
			SetBody(body)

		resp, err := req.Post("/api/user/orders")

		noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на загрузку заказа в систему лояльности")
		validStatus := suite.Assert().Equalf(http.StatusAccepted, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)

		if !noRespErr || !validStatus {
			dump := dumpRequest(suite.T(), req.RawRequest, bytes.NewReader(body))
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("await_order_processed", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				suite.T().Errorf("Не удалось дождаться окончания расчета начисления")
				return
			case <-ticker.C:
				var orders []order

				ctx, cancel := context.WithTimeout(ctx, time.Second)

				req := httpc.R().
					SetContext(ctx).
					SetResult(&orders)

				resp, err := req.Get("/api/user/orders")
				cancel()

				noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на получение статуса расчета начисления в системе лояльности")
				validStatus := suite.Assert().Containsf([]int{http.StatusOK, http.StatusNoContent}, resp.StatusCode(),
					"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
				)
				validContentType := suite.Assert().Containsf(resp.Header().Get("Content-Type"), "application/json",
					"Заголовок ответа Content-Type содержит несоответствующее значение",
				)

				if !noRespErr || !validStatus || !validContentType {
					dump := dumpRequest(suite.T(), req.RawRequest, nil)
					suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
					continue
				}

				// wait for miracle
				if resp.StatusCode() != http.StatusOK || len(orders) == 0 ||
					orders[0].Status != "PROCESSED" {
					continue
				}

				o := orders[0]
				suite.Assert().Equal(orderNum, o.Number, "Номер заказа не соответствует ожидаемому")
				suite.Assert().Equal("PROCESSED", o.Status, "Статус заказа не соответствует ожидаемому")
				suite.Assert().Equal(expectedAccrual, o.Accrual, "Начисление за заказ не соответствует ожидаемому")

				return
			}
		}
	})

	suite.Run("check_balance", func() {
		var balance userBalance

		req := httpc.R().
			SetResult(&balance)

		resp, err := req.Get("/api/user/balance")

		noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на проверку баланса пользователя в системе лояльности")
		validStatus := suite.Assert().Containsf([]int{http.StatusOK, http.StatusNoContent}, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)
		validContentType := suite.Assert().Containsf(resp.Header().Get("Content-Type"), "application/json",
			"Заголовок ответа Content-Type содержит несоответствующее значение",
		)

		expected := userBalance{
			Current:   expectedAccrual,
			Withdrawn: 0,
		}

		validBalance := suite.Assert().Equal(expected, balance, "Баланс пользователя не соответствует ожидаемому")

		if !noRespErr || !validStatus || !validBalance || !validContentType {
			dump := dumpRequest(suite.T(), req.RawRequest, nil)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
			return
		}
	})

	withdrawOrder, err := generateOrderNumber(suite.T())
	suite.Require().NoError(err, "Не удалось сгенерировать номер заказа")

	randSrc := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	min, max := 100, 700
	withdrawSum := float32(randSrc.Intn(max-min)+min) + randSrc.Float32()

	suite.Run("withdraw_balance", func() {
		body := []byte(`{
			"order": "` + withdrawOrder + `",
    		"sum": ` + strconv.FormatFloat(float64(withdrawSum), 'f', 2, 32) + `
		}`)

		req := httpc.R().
			SetHeader("Content-Type", "application/json").
			SetBody(body)

		resp, err := req.Post("/api/user/balance/withdraw")

		noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на списание средств с баланса пользователя в системе лояльности")
		validStatus := suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)

		if !noRespErr || !validStatus {
			dump := dumpRequest(suite.T(), req.RawRequest, bytes.NewReader(body))
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
			return
		}
	})

	suite.Run("recheck_balance", func() {
		var balance userBalance

		req := httpc.R().
			SetResult(&balance)

		resp, err := req.Get("/api/user/balance")

		noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на перепроверку баланса пользователя в системе лояльности")
		validStatus := suite.Assert().Equal(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)
		validContentType := suite.Assert().Containsf(resp.Header().Get("Content-Type"), "application/json",
			"Заголовок ответа Content-Type содержит несоответствующее значение",
		)

		expectedBalance := expectedAccrual - withdrawSum
		expectedWithdrawn := withdrawSum
		validBalance := suite.Assert().InDelta(expectedBalance, balance.Current, 0.01, "Баланс пользователя не соответствует ожидаемому")
		validWithdrawn := suite.Assert().InDelta(expectedWithdrawn, balance.Withdrawn, 0.01, "Списанная сумма не соответствует ожидаемой")

		if !noRespErr || !validStatus || !validBalance || !validWithdrawn || !validContentType {
			dump := dumpRequest(suite.T(), req.RawRequest, nil)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
			return
		}
	})

	suite.Run("check_withdrawals", func() {
		var withdrawals []userWithdrawal

		req := httpc.R().
			SetHeader("Content-Type", "application/json").
			SetResult(&withdrawals)

		resp, err := req.Get("/api/user/withdrawals")

		noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на получение списаний пользователя в системе лояльности")
		validStatus := suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)

		if !noRespErr || !validStatus {
			dump := dumpRequest(suite.T(), req.RawRequest, nil)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
			return
		}

		suite.Require().NotEmpty(withdrawals)
		suite.Assert().Equal(withdrawOrder, withdrawals[0].Order, "Номер заказа не совпадает с ожидаемым")
		suite.Assert().InDelta(withdrawSum, withdrawals[0].Sum, 0.01, "Списанная сумма не соответствует ожидаемой")
	})
}
