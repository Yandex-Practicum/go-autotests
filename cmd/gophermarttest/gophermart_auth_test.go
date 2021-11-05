package main

import (
	"bytes"
	"net/http"

	"github.com/go-resty/resty/v2"

	"github.com/Yandex-Practicum/go-autotests/internal/random"
)

// TestUserAuth checks registration and authentication
func (suite *GophermartSuite) TestUserAuth() {
	login := random.ASCIIString(4, 10)
	password := random.ASCIIString(16, 32)

	suite.Run("register_user", func() {
		m := []byte(`
			{
				"login": "` + login + `",
				"password": "` + password + `"
			}
		`)

		req := resty.New().
			SetHostURL(suite.gophermartServerAddress).
			R().
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
	})

	suite.Run("login_user", func() {
		m := []byte(`
			{
				"login": "` + login + `",
				"password": "` + password + `"
			}
		`)

		req := resty.New().
			SetHostURL(suite.gophermartServerAddress).
			R().
			SetHeader("Content-Type", "application/json").
			SetBody(m)

		resp, err := req.Post("/api/user/login")

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
	})
}
