package main

import (
	"net/http"
	"strconv"
	"testing"
)

func TestIteration3A(t *testing.T) {
	t.Run("TestCounter", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			e := New(t)
			c := ClientForDefaultServer(e)

			resp, err := c.R().Post("update/counter/test1/1")
			e.NoError(err, "Ошибка при выполнении запроса к серверу")
			e.Equal(http.StatusOK, resp.StatusCode(), "При добавлении нового значения сервер должен вернуть код 200 (http.StatusOK)")

			resp, err = c.R().Get("value/counter/test1")
			e.NoError(err, "Ошибка при выполнении запроса к серверу")
			e.Equalf(http.StatusOK, resp.StatusCode(), "При получении известного значения метрики сервер должен вернуть код 200 (http.StatusOK)")

			val, err := strconv.Atoi(string(resp.Body()))
			e.NoErrorf(err, "Ошибка при попытке распарсить значение %q", resp.Body())
			e.Equal(1, val, "Сервер вернул не то значение, которое было отправлено")

			resp, err = c.R().Post("update/counter/test1/2")
			e.NoError(err, "Ошибка при выполнении запроса к серверу")
			e.Equal(http.StatusOK, resp.StatusCode(), "При добавлении нового значения сервер должен вернуть код 200 (http.StatusOK)")

			resp, err = c.R().Get("value/counter/test1")
			e.NoError(err, "Ошибка при выполнении запроса к серверу")
			val, err = strconv.Atoi(string(resp.Body()))
			e.NoErrorf(err, "Ошибка при попытке распарсить обновлённое значение %q", resp.Body())
			e.Equal(3, val, "Сервер должен был вернуть сумму отправленных значений: 1+2")

			// добавляем второе значение
			resp, err = c.R().Post("update/counter/test2/4")
			e.NoError(err, "Ошибка при выполнении запроса к серверу")
			e.Equal(http.StatusOK, resp.StatusCode(), "При добавлении нового значения сервер должен вернуть код 200 (http.StatusOK)")

			resp, err = c.R().Get("value/counter/test2")
			e.NoError(err, "Ошибка при выполнении запроса к серверу")
			e.Equalf(http.StatusOK, resp.StatusCode(), "При получении известного значения метрики сервер должен вернуть код 200 (http.StatusOK)")

			val, err = strconv.Atoi(string(resp.Body()))
			e.NoErrorf(err, "Ошибка при попытке распарсить значение %q", resp.Body())
			e.Equal(4, val, "Сервер вернул не то значение, которое было отправлено для второй метрики")

			resp, err = c.R().Get("value/counter/test1")
			e.NoError(err, "Ошибка при выполнении запроса к серверу")
			val, err = strconv.Atoi(string(resp.Body()))
			e.NoErrorf(err, "Ошибка при попытке распарсить обновлённое значение %q", resp.Body())
			e.Equal(3, val, "Сервер неправильно вернул значение первого счётчика")
		})

		t.Run("unknown-value", func(t *testing.T) {
			e := New(t)
			c := ClientForDefaultServer(e)
			resp, err := c.R().Get("value/counter/unknown")
			e.NoError(err, "Ошибка при выполнении запроса к серверу")
			e.Equal(http.StatusNotFound, resp.StatusCode(), "При запросе неизвестного значения должен возвращаться код 404 (http.StatusNotFound)")
		})
	})

	t.Run("TestGauge", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			e := New(t)
			c := ClientForDefaultServer(e)

			resp, err := c.R().Post("update/gauge/test1/1.5")
			e.NoError(err, "Ошибка при выполнении запроса к серверу")
			e.Equal(http.StatusOK, resp.StatusCode(), "При добавлении нового значения сервер должен вернуть код 200 (http.StatusOK)")

			resp, err = c.R().Get("value/gauge/test1")
			e.NoError(err, "Ошибка при выполнении запроса к серверу")
			e.Equalf(http.StatusOK, resp.StatusCode(), "При получении известного значения метрики сервер должен вернуть код 200 (http.StatusOK)")

			val, err := strconv.ParseFloat(string(resp.Body()), 64)
			e.NoErrorf(err, "Ошибка при попытке распарсить значение %q", resp.Body())
			e.InEpsilon(1.5, val, 0.1, "Сервер вернул не то значение, которое было отправлено")

			resp, err = c.R().Post("update/gauge/test1/2")
			e.NoError(err, "Ошибка при выполнении запроса к серверу")
			e.Equal(http.StatusOK, resp.StatusCode(), "При добавлении нового значения сервер должен вернуть код 200 (http.StatusOK)")

			resp, err = c.R().Get("value/gauge/test1")
			e.NoError(err, "Ошибка при выполнении запроса к серверу")
			val, err = strconv.ParseFloat(string(resp.Body()), 64)
			e.NoErrorf(err, "Ошибка при попытке распарсить обновлённое значение %q", resp.Body())
			e.InEpsilon(2, val, 0.1, "Сервер вернул не то значение, которое было отправлено при обновлении существующего значения")

			// добавляем второе значение
			resp, err = c.R().Post("update/gauge/test2/4")
			e.NoError(err, "Ошибка при выполнении запроса к серверу")
			e.Equal(http.StatusOK, resp.StatusCode(), "При добавлении нового значения сервер должен вернуть код 200 (http.StatusOK)")

			resp, err = c.R().Get("value/gauge/test2")
			e.NoError(err, "Ошибка при выполнении запроса к серверу")
			e.Equalf(http.StatusOK, resp.StatusCode(), "При получении известного значения метрики сервер должен вернуть код 200 (http.StatusOK)")

			val, err = strconv.ParseFloat(string(resp.Body()), 64)
			e.NoErrorf(err, "Ошибка при попытке распарсить значение %q", resp.Body())
			e.InEpsilon(4, val, 0.1, "Сервер вернул не то значение, которое было отправлено для второй метрики")

			resp, err = c.R().Get("value/gauge/test1")
			e.NoError(err, "Ошибка при выполнении запроса к серверу")
			val, err = strconv.ParseFloat(string(resp.Body()), 64)
			e.NoErrorf(err, "Ошибка при попытке распарсить обновлённое значение %q", resp.Body())
			e.InEpsilon(2, val, 0.1, "Сервер неправильно вернул значение первого счётчика")
		})

		t.Run("unknown-value", func(t *testing.T) {
			e := New(t)
			c := ClientForDefaultServer(e)
			resp, err := c.R().Get("value/gauge/unknown")
			e.NoError(err, "Ошибка при выполнении запроса к серверу")
			e.Equal(http.StatusNotFound, resp.StatusCode(), "При запросе неизвестного значения должен возвращаться код 404 (http.StatusNotFound)")
		})
	})
}
