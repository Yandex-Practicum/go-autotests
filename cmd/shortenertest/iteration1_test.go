package main

import (
	"crypto/rand"
	"errors"
	"math/big"
	mathrand "math/rand"
	"net/http"
	"net/url"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIteration1 tests that:
// - http server is alive
// - handlers conforms lesson API requirements
func TestIteration1(t *testing.T) {
	// create HTTP client without redirects support
	errRedirectBlocked := errors.New("HTTP redirect blocked")
	httpc := resty.New().
		SetRedirectPolicy(resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
			return errRedirectBlocked
		}),
	)

	// shorten URL
	targetURL := generateTestURL(t)

	resp, err := httpc.R().
		SetBody(targetURL).
		Post(config.TargetAddress)
	if !assert.NoError(t, err) {
		return
	}

	shortenURL := string(resp.Body())

	assert.Equal(t, http.StatusCreated, resp.StatusCode())
	assert.NoError(t, func() error {
		_, err := url.Parse(shortenURL)
		return err
	}())

	// expand URL
	resp, err = httpc.R().Get(shortenURL)
	if !errors.Is(err, errRedirectBlocked) && !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode())
	assert.Equal(t, targetURL, resp.Header().Get("Location"))
}

func generateTestURL(t *testing.T) string {
	// generate PROTO
	proto := "http://"
	if mathrand.Float32() < 0.5 {
		proto = "https://"
	}

	// generate DOMAIN
	var letters = "0123456789abcdefghijklmnopqrstuvwxyz"

	minLen, maxLen := 5, 15
	domainLen := mathrand.Intn(maxLen - minLen) + minLen

	lettersLen := big.NewInt(int64(len(letters)))

	ret := make([]byte, domainLen)
	for i := 0; i < domainLen; i++ {
		num, err := rand.Int(rand.Reader, lettersLen)
		require.NoError(t, err)
		ret[i] = letters[num.Int64()]
	}
	domain := string(ret)

	// generate ZONE
	var zones = []string{".com", ".ru", ".net", ".biz", ".yandex"}
	zone := zones[mathrand.Intn(len(zones))]

	return proto + domain + zone
}