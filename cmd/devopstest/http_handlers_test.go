package main

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
)

// TestPostHandlers tests that:
// - http server is alive
// - server exposes post handlers with mandatory args
func TestPostHandlers(t *testing.T) {
	// create HTTP client without redirects support
	errRedirectBlocked := errors.New("HTTP redirect blocked")
	httpc := resty.New().
		SetRedirectPolicy(resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
			return errRedirectBlocked
		}),
		)

	resp, err := httpc.R().
		Post(config.TargetAddress + "?id=githubActionGauge&value=100&type=gauge")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode(), "invalid post gauge metrics")

	resp, err = httpc.R().
		Post(config.TargetAddress + "?value=100&type=gauge")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode(), "gauge without id")

	resp, err = httpc.R().
		Post(config.TargetAddress + "?id=githubActionGauge&type=gauge")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode(), "gauge without value")

	resp, err = httpc.R().
		Post(config.TargetAddress + "?id=githubActionCounter&value=100&type=counter")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode(), "invalid post counter metrics")

	resp, err = httpc.R().
		Post(config.TargetAddress + "?value=100&type=clounter")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode(), "clounter without id")

	resp, err = httpc.R().
		Post(config.TargetAddress + "?id=githubActionCounter&type=clounter")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode(), "clounter without value")

	resp, err = httpc.R().
		Post(config.TargetAddress + "?id=githubActionCounter&value=100&type=unknownType")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotImplemented, resp.StatusCode(), "invalid type")
}

// TestGetHandlers tests that:
// - http server is alive
// - server exposes post handlers with mandatory args
func TestGetHandlers(t *testing.T) {
	// create HTTP client without redirects support
	errRedirectBlocked := errors.New("HTTP redirect blocked")
	httpc := resty.New().
		SetRedirectPolicy(resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
			return errRedirectBlocked
		}),
		)

	resp, err := httpc.R().
		Get(config.TargetAddress)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode(), "invalid get metrics")
}

// TestPostHandlers2 tests that:
// - http server is alive
// - server exposes post handlers with mandatory args
func TestPostHandlers2(t *testing.T) {
	// create HTTP client without redirects support
	errRedirectBlocked := errors.New("HTTP redirect blocked")
	httpc := resty.New().
		SetRedirectPolicy(resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
			return errRedirectBlocked
		}),
		)

	resp, err := httpc.R().
		Post(config.TargetAddress + "/update/gauge/githubActionGauge/100")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode(), "invalid post gauge metrics")

	resp, err = httpc.R().
		Post(config.TargetAddress + "/update/counter/githubActionCounter/100")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode(), "invalid post counter metrics")

	resp, err = httpc.R().
		Post(config.TargetAddress + "/update/unknown/githubActionCounter/100")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotImplemented, resp.StatusCode(), "invalid post type of metrics")

	resp, err = httpc.R().
		Post(config.TargetAddress + "/updater/counter/githubActionCounter/100")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode(), "invalid method")

	resp, err = httpc.R().
		Post(config.TargetAddress + "/update/counter/githubActionCounter/")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode(), "invalid type value")
}

// TestUpdateGetMetrics tests that:
// - http server is alive
// - post data on server and get expected value
func TestUpdateGetMetrics(t *testing.T) {
	// create HTTP client without redirects support
	errRedirectBlocked := errors.New("HTTP redirect blocked")
	httpc := resty.New().
		SetRedirectPolicy(resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
			return errRedirectBlocked
		}),
		)

	updateGauge := config.TargetAddress + "/update/gauge/githubSetGet/"
	valueValue := config.TargetAddress + "/value/gauge/githubSetGet"
	updateCounter := config.TargetAddress + "/update/counter/githubSetGet/"
	valueCounter := config.TargetAddress + "/value/counter/githubSetGet"

	count := 100
	t.Run("gauge", func(t *testing.T) {
		for i := 0; i < count; i++ {
			v := fmt.Sprintf("%.3f", rand.Float64()*1000000)
			resp, err := httpc.R().
				Post(updateGauge + v)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode(), "invalid post gauge metrics")

			resp, err = httpc.R().
				Get(valueValue)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode(), "invalid get gauge metrics")

			assert.Equal(t, v, resp.String())
		}
	})
	t.Run("counter", func(t *testing.T) {
		var a int64 = 0
		resp, err := httpc.R().
			Get(valueCounter)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode(), "invalid get counter metrics")
		a, _ = strconv.ParseInt(resp.String(), 0, 64)

		for i := 0; i < count; i++ {
			v := rand.Intn(1024)
			a += int64(v)
			resp, err = httpc.R().
				Post(updateCounter + strconv.Itoa(v))
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode(), "invalid post counter metrics")

			resp, err = httpc.R().
				Get(valueCounter)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode(), "invalid get counter metrics")

			assert.Equal(t, fmt.Sprintf("%d", a), resp.String())
		}
	})
}

// TestMonitoringData tests that:
// - http monitor is alive
// - poll server every 100ms
// - get metrics from server posted by monitor
func TestMonitoringData(t *testing.T) {
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
		value  string
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
		// t.Log("tick", n)
		for i, tt := range tests {
			if tt.ok {
				continue
			}
			var (
				resp *resty.Response
				err  error
			)
			time.Sleep(100 * time.Millisecond)
			switch tt.method {
			case "gauge":
				resp, err = httpc.R().
					Get(config.TargetAddress + "/value/gauge/" + tt.name)
			case "counter":
				resp, err = httpc.R().
					Get(config.TargetAddress + "/value/counter/" + tt.name)
			}
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode(), "invalid get metrics")

			if tt.value == "" {
				// t.Logf("Get data of %v, was: %v, now: %v\n", tt.name, tt.value, resp.String())
				tests[i].value = resp.String()
			}
			if tt.value != "" && (tt.value != resp.String() || tt.static) {
				// t.Logf("Updated data of %v, was: %v, now: %v\n", tt.name, tt.value, resp.String())
				tests[i].ok = true
				ok++
				continue
			}
		}
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, tt.ok)
		})
	}
}
