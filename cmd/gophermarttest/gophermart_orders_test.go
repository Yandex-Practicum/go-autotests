package main

import (
	"bytes"
	"net/http"
	"net/http/cookiejar"

	"github.com/go-resty/resty/v2"

	"github.com/Yandex-Practicum/go-autotests/internal/random"
)

// TestUserOrders checks order upload and status response
func (suite *GophermartSuite) TestUserOrders() {
	jar, err := cookiejar.New(nil)
	suite.Require().NoError(err, "Не удалось создать объект cookie jar")

	httpc := resty.New().
		SetHostURL(suite.gophermartServerAddress).
		SetCookieJar(jar)

	login := random.ASCIIString(7, 14)
	password := random.ASCIIString(16, 32)

	orderNum, err := generateOrderNumber(suite.T())
	suite.Require().NoError(err, "Не удалось сгенерировать номер заказа")

	suite.Run("unauthorized_order_upload", func() {
		number, err := generateOrderNumber(suite.T())
		suite.Require().NoError(err, "Не удалось сгенерировать номер заказа")

		body := []byte(number)

		req := httpc.R().
			SetHeader("Content-Type", "text/plain").
			SetBody(body)

		resp, err := req.Post("/api/user/orders")

		noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на регистрацию механики")
		validStatus := suite.Assert().Equalf(http.StatusUnauthorized, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)

		if !noRespErr || !validStatus {
			dump := dumpRequest(suite.T(), req.RawRequest, bytes.NewReader(body))
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("unauthorized_orders_list", func() {
		req := httpc.R()
		resp, err := req.Get("/api/user/orders")

		noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на регистрацию механики")
		validStatus := suite.Assert().Equalf(http.StatusUnauthorized, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)

		if !noRespErr || !validStatus {
			dump := dumpRequest(suite.T(), req.RawRequest, nil)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("register_user", func() {
		m := []byte(`{"login": "` + login + `","password": "` + password + `"}`)

		req := httpc.R().
			SetHeader("Content-Type", "application/json").
			SetBody(m)

		resp, err := req.Post("/api/user/register")

		noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на регистрацию механики")
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

	suite.Run("bad_order_upload", func() {
		body := []byte(`12345678902`)

		req := httpc.R().
			SetHeader("Content-Type", "text/plain").
			SetBody(body)

		resp, err := req.Post("/api/user/orders")

		noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на регистрацию механики")
		validStatus := suite.Assert().Equalf(http.StatusUnprocessableEntity, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)

		if !noRespErr || !validStatus {
			dump := dumpRequest(suite.T(), req.RawRequest, bytes.NewReader(body))
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("order_upload", func() {
		body := []byte(orderNum)

		req := httpc.R().
			SetHeader("Content-Type", "text/plain").
			SetBody(body)

		resp, err := req.Post("/api/user/orders")

		noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на регистрацию механики")
		validStatus := suite.Assert().Equalf(http.StatusAccepted, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)

		if !noRespErr || !validStatus {
			dump := dumpRequest(suite.T(), req.RawRequest, bytes.NewReader(body))
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("duplicate_order_upload_same_user", func() {
		body := []byte(orderNum)

		req := httpc.R().
			SetHeader("Content-Type", "text/plain").
			SetBody(body)

		resp, err := req.Post("/api/user/orders")

		noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на регистрацию механики")
		validStatus := suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)

		if !noRespErr || !validStatus {
			dump := dumpRequest(suite.T(), req.RawRequest, bytes.NewReader(body))
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("orders_list", func() {
		var orders []order

		req := httpc.R().
			SetResult(&orders)

		resp, err := req.Get("/api/user/orders")

		noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на регистрацию механики")
		validStatus := suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
		)
		validContentType := suite.Assert().Containsf(resp.Header().Get("Content-Type"), "application/json",
			"Заголовок ответа Content-Type содержит несоответствующее значение",
		)

		validOrdersLen := suite.Assert().Len(orders, 1, "Ожидаем отличное от полученного кол-во заказов")
		validOrderNum := validOrdersLen && suite.Assert().Equal(orderNum, orders[0].Number,
			"Ожидаем другой номер заказа в ответе")
		validOrderStatus := validOrdersLen && suite.Assert().Contains([]string{"NEW", "PROCESSING"}, orders[0].Status,
			"Ожидаем другой статус заказа в ответе")

		if !noRespErr || !validStatus || !validOrdersLen ||
			!validOrderNum || !validOrderStatus || !validContentType {
			dump := dumpRequest(suite.T(), req.RawRequest, nil)
			suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
		}
	})

	suite.Run("duplicate_order_upload_other_user", func() {
		djar, err := cookiejar.New(nil)
		suite.Require().NoError(err, "Не удалось создать объект cookie jar")

		dhttpc := resty.New().
			SetHostURL(suite.gophermartServerAddress).
			SetCookieJar(djar)

		// register new user
		{
			dlogin := random.ASCIIString(7, 14)
			dpassword := random.ASCIIString(16, 32)
			m := []byte(`{"login": "` + dlogin + `","password": "` + dpassword + `"}`)

			req := dhttpc.R().
				SetHeader("Content-Type", "application/json").
				SetBody(m)

			resp, err := req.Post("/api/user/register")

			noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на регистрацию механики")
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

			if authHeader != "" {
				dhttpc.SetHeader("Authorization", authHeader)
			}
		}

		// upload duplicate order
		{
			body := []byte(orderNum)

			req := dhttpc.R().
				SetHeader("Content-Type", "text/plain").
				SetBody(body)

			resp, err := req.Post("/api/user/orders")

			noRespErr := suite.Assert().NoErrorf(err, "Ошибка при попытке сделать запрос на регистрацию механики")
			validStatus := suite.Assert().Equalf(http.StatusConflict, resp.StatusCode(),
				"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL,
			)

			if !noRespErr || !validStatus {
				dump := dumpRequest(suite.T(), req.RawRequest, bytes.NewReader(body))
				suite.T().Logf("Оригинальный запрос:\n\n%s", dump)
			}
		}
	})
}
