package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/go-resty/resty/v2"
)

func TestIteration1(t *testing.T) {
	t.Run("TestCounterHandlers", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			e := New(t)
			c := ClientForDefaultServer(e)
			req := c.R()
			resp, err := req.Post("update/counter/testGauge/100")
			e.Require.NoError(err, "Ошибка при выполнении запроса")
			e.Equal(http.StatusOK, resp.StatusCode(), "При добавлении нового значения сервер должен возвращать код 200 (http.StatusOK)")

			req = c.R()
			resp, err = req.Post("update/counter/testGauge/101")
			e.Require.NoError(err, "Ошибка при выполнении запроса")
			e.Equal(http.StatusOK, resp.StatusCode(), "При обновлении значения сервер должен возвращать код 200 (http.StatusOK)")
		})

		t.Run("without-id", func(t *testing.T) {
			e := New(t)
			c := ClientForDefaultServer(e)
			req := c.R()

			resp, err := req.Post("update/counter/testGauge/")
			e.Require.NoError(err, "Ошибка при выполнении запроса")
			e.Contains([]int{http.StatusBadRequest, http.StatusNotFound}, resp.StatusCode(),
				"При попытке обновления значения без ID - сервер должен вернуть ошибку 400 или 404 (http.StatusBadRequest, http.StatusNotFound).")
		})

		t.Run("bad value", func(t *testing.T) {
			e := New(t)
			c := ClientForDefaultServer(e)
			req := c.R()

			resp, err := req.Post("update/gauge/testGauge/bad-value")
			e.Require.NoError(err, "Ошибка при выполнении запроса")
			e.Equal(http.StatusBadRequest, resp.StatusCode(), "При получении неправильного значения сервер должен вернуть ошибку 400 (http.StatusBadRequest)")
		})

	})

	t.Run("TestGaugeHandlers", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			e := New(t)
			c := ClientForDefaultServer(e)
			req := c.R()
			resp, err := req.Post("update/gauge/testGauge/100")
			e.Require.NoError(err, "Ошибка при выполнении запроса")
			e.Equal(http.StatusOK, resp.StatusCode(), "При добавлении нового значения сервер должен возвращать код http.StatusOK (200)")

			req = c.R()
			resp, err = req.Post("update/gauge/testGauge/101")
			e.Require.NoError(err, "Ошибка при выполнении запроса")
			e.Equal(http.StatusOK, resp.StatusCode(), "При обновлении значения сервер должен возвращать код http.StatusOK (200)")
		})

		t.Run("without-id", func(t *testing.T) {
			e := New(t)
			c := ClientForDefaultServer(e)
			req := c.R()

			resp, err := req.Post("update/gauge/testGauge/")
			e.Require.NoError(err, "Ошибка при выполнении запроса")
			e.Contains([]int{http.StatusBadRequest, http.StatusNotFound}, resp.StatusCode(),
				"При попытке обновления значения без ID - сервер должен вернуть ошибку http.StatusBadRequest или http.StatusNotFound  (400, 404).")
		})

		t.Run("bad value", func(t *testing.T) {
			e := New(t)
			c := ClientForDefaultServer(e)
			req := c.R()

			resp, err := req.Post("update/gauge/testGauge/bad-value")
			e.Require.NoError(err, "Ошибка при выполнении запроса")
			e.Equal(http.StatusBadRequest, resp.StatusCode(), "При получении неправильного значения сервер должен вернуть ошибку http.StatusBadRequest (400)")
		})
	})

	t.Run("unexpected path", func(t *testing.T) {
		e := New(t)
		c := ClientForDefaultServer(e)
		req := c.R()

		for _, path := range []string{"unknown-path", "unknown-path/gauge/testGauge/100"} {
			resp, err := req.Post(path)
			e.Require.NoError(err, "Ошибка при выполнении запроса")
			e.Equal(http.StatusNotFound, resp.StatusCode(), "При получении запроса к неизвестному пути сервер должен возвращать ошибку http.StatusNotFound (404)")
		}
	})

	t.Run("unknown-metric-type", func(t *testing.T) {
		e := New(t)
		c := ClientForDefaultServer(e)
		req := c.R()
		resp, err := req.Post("update/unknown/testGauge/100")
		e.Require.NoError(err, "Ошибка при выполнении запроса")
		e.Contains([]int{http.StatusBadRequest, http.StatusNotFound}, resp.StatusCode(),
			"При попытке обновления метрики неизвестного типа сервер должен вернуть ошибку http.StatusBadRequest или http.StatusNotFound (400, 404)")
	})
}

func ClientForDefaultServer(e *Env) *resty.Client {
	StartProcessWhichListenPort(e, serverDefaultHost, serverDefaultPort, "metric server", ServerFilePath(e))
	address := fmt.Sprintf("http://%v:%v", serverDefaultHost, serverDefaultPort)
	return RestyClient(e, address)
}
