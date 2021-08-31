package main

import (
	"errors"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`            // Параметр кодирую строкой, принося производительность в угоду наглядности.
	Delta *int64   `json:"delta,omitempty"` //counter
	Value *float64 `json:"value,omitempty"` //gauge
}

// TestJsonPostHandlers tests that:
// - http server is alive
// - server exposes post handlers with mandatory args
func TestJsonPostHandlers(t *testing.T) {
	// create HTTP client without redirects support
	errRedirectBlocked := errors.New("HTTP redirect blocked")
	httpc := resty.New().
		SetRedirectPolicy(resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
			return errRedirectBlocked
		}),
		)

	value := float64(100)
	delta := int64(256)
	resp, err := httpc.R().
		SetHeader("Content-Type", "application/json").
		SetBody(&Metrics{
			ID:    "githubActionGauge",
			MType: "gauge",
			Value: &value,
		}).
		Post(config.TargetAddress + "/update/")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode(), "invalid post gauge metrics")

	resp, err = httpc.R().
		SetHeader("Content-Type", "application/json").
		SetBody(&Metrics{
			ID:    "githubActionCounter",
			MType: "counter",
			Delta: &delta,
		}).
		Post(config.TargetAddress + "/update/")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode(), "invalid post counter metrics")

	resp, err = httpc.R().
		SetHeader("Content-Type", "application/json").
		SetBody(&Metrics{
			ID:    "githubActionCounter",
			MType: "unknown",
			Delta: &delta,
		}).
		Post(config.TargetAddress + "/update/")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotImplemented, resp.StatusCode(), "invalid post type of metrics")

	resp, err = httpc.R().
		SetHeader("Content-Type", "application/json").
		SetBody(&Metrics{
			ID:    "githubActionCounter",
			MType: "counter",
			Delta: &delta,
		}).
		Post(config.TargetAddress + "/updater")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode(), "invalid method")

	resp, err = httpc.R().
		SetHeader("Content-Type", "application/json").
		SetBody(&Metrics{
			ID:    "githubActionCounter",
			MType: "counter",
		}).
		Post(config.TargetAddress + "/updater")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode(), "invalid type value")
}

// TestUpdateGetMetrics tests that:
// - http server is alive
// - post data on server and get expected value
func TestJsonUpdateGetMetrics(t *testing.T) {
	// create HTTP client without redirects support
	errRedirectBlocked := errors.New("HTTP redirect blocked")
	httpc := resty.New().
		SetRedirectPolicy(resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
			return errRedirectBlocked
		}),
		)

	count := 100
	t.Run("gauge", func(t *testing.T) {
		id := "githubSetGetGauge"
		for i := 0; i < count; i++ {
			value := rand.Float64() * 1000000
			resp, err := httpc.R().
				SetHeader("Content-Type", "application/json").
				SetBody(&Metrics{
					ID:    id,
					MType: "gauge",
					Value: &value,
				}).
				Post(config.TargetAddress + "/update/")
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode(), "invalid post gauge metrics")

			var result Metrics
			resp, err = httpc.R().
				SetHeader("Content-Type", "application/json").
				SetBody(&Metrics{
					ID:    id,
					MType: "gauge",
					Value: &value,
				}).
				SetResult(&result).
				Post(config.TargetAddress + "/value/")
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode(), "invalid get gauge metrics")
			assert.Nil(t, result.Delta, "Result Delta is not empty")
			require.NotNil(t, result.Value, "Result Value is empty")
			assert.Equal(t, value, *result.Value)
		}
	})
	t.Run("counter", func(t *testing.T) {
		id := "githubSetGetCounter4"
		var result Metrics
		var acc int64
		resp, err := httpc.R().
			SetHeader("Content-Type", "application/json").
			SetBody(&Metrics{
				ID:    id,
				MType: "counter",
				Delta: &acc,
			}).
			SetResult(&result).
			Post(config.TargetAddress + "/value/")
		assert.NoError(t, err)

		switch resp.StatusCode() {
		case http.StatusOK:
			assert.Nil(t, result.Value, "Result Value is not empty")
			require.NotNil(t, result.Delta, "Result Delta is empty")
			acc = *result.Delta
		case http.StatusNotFound:
			acc = 0
		default:
			t.Fatalf("invalid status code: %d", resp.StatusCode())
		}

		for i := 0; i < count; i++ {
			value := int64(rand.Intn(1024))
			acc += value
			resp, err = httpc.R().
				SetHeader("Content-Type", "application/json").
				SetBody(&Metrics{
					ID:    id,
					MType: "counter",
					Delta: &value,
				}).
				Post(config.TargetAddress + "/update/")
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode(), "invalid post counter metrics")

			resp, err = httpc.R().
				SetHeader("Content-Type", "application/json").
				SetBody(&Metrics{
					ID:    id,
					MType: "counter",
				}).
				SetResult(&result).
				Post(config.TargetAddress + "/value/")
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode(), "invalid get counter metrics")
			assert.Nil(t, result.Value, "Result Value is not empty")
			require.NotNil(t, result.Delta, "Result Delta is empty")
			require.Equal(t, acc, *result.Delta, resp.String())
		}
	})
}

func TestJsonMonitoringData(t *testing.T) {
	// create HTTP client without redirects support
	errRedirectBlocked := errors.New("HTTP redirect blocked")
	httpc := resty.New().
		SetRedirectPolicy(resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
			return errRedirectBlocked
		}),
		)

	tests := []struct {
		name   string
		method string
		value  float64
		delta  int64
		update int
		ok     bool
		static bool
	}{
		{method: "counter", name: "PollCount"},
		{method: "gauge", name: "Alloc"},
		{method: "gauge", name: "BuckHashSys", static: true},
		{method: "gauge", name: "Frees"},
		{method: "gauge", name: "GCCPUFraction", static: true},
		{method: "gauge", name: "GCSys", static: true},
		{method: "gauge", name: "HeapAlloc"},
		{method: "gauge", name: "HeapIdle"},
		{method: "gauge", name: "HeapInuse"},
		{method: "gauge", name: "HeapObjects"},
		{method: "gauge", name: "HeapReleased"},
		{method: "gauge", name: "HeapSys", static: true},
		{method: "gauge", name: "LastGC", static: true},
		{method: "gauge", name: "Lookups", static: true},
		{method: "gauge", name: "MCacheInuse", static: true},
		{method: "gauge", name: "MCacheSys", static: true},
		{method: "gauge", name: "MSpanInuse", static: true},
		{method: "gauge", name: "MSpanSys", static: true},
		{method: "gauge", name: "Mallocs"},
		{method: "gauge", name: "NextGC", static: true},
		{method: "gauge", name: "NumForcedGC", static: true},
		{method: "gauge", name: "NumGC", static: true},
		{method: "gauge", name: "OtherSys", static: true},
		{method: "gauge", name: "PauseTotalNs", static: true},
		{method: "gauge", name: "StackInuse", static: true},
		{method: "gauge", name: "StackSys", static: true},
		{method: "gauge", name: "Sys", static: true},
		{method: "gauge", name: "TotalAlloc"},
	}
	for ok, n := 0, 0; n < 120 && ok != len(tests); n++ {
		t.Log("tick", n, len(tests)-ok)
		for i, tt := range tests {
			if tt.ok {
				continue
			}
			var (
				resp *resty.Response
				err  error
			)
			time.Sleep(100 * time.Millisecond)

			var result Metrics
			resp, err = httpc.R().
				SetHeader("Content-Type", "application/json").
				SetBody(&Metrics{
					ID:    tt.name,
					MType: tt.method,
				}).
				SetResult(&result).
				Post(config.TargetAddress + "/value/")
			assert.NoError(t, err)

			switch resp.StatusCode() {
			case http.StatusOK:
				// t.Log("get", tt.name)
			case http.StatusNotFound:
				// t.Log("new", tt.name)
				continue
			default:
				t.Fatalf("invalid status code: %d", resp.StatusCode())
			}
			assert.Equal(t, http.StatusOK, resp.StatusCode(), "invalid get metrics")

			switch tt.method {
			case "gauge":
				require.NotNil(t, result.Value, "Result Value is empty")
				// t.Log("tock", tt.value, *result.Value)
				if (tt.update != 0 && *result.Value != tt.value) || tt.static {
					tests[i].ok = true
					ok++
					t.Logf("get %s, %s", tt.method, tt.name)
				}
				tests[i].value = *result.Value
			case "counter":
				require.NotNil(t, result.Delta, "Result Delta is empty")
				if (tt.update != 0 && *result.Delta != tt.delta) || tt.static {
					tests[i].ok = true
					ok++
					t.Logf("get %s, %s", tt.method, tt.name)
				}
				tests[i].delta = *result.Delta
			}
			tests[i].update++
		}
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, tt.ok)
		})
	}
}
