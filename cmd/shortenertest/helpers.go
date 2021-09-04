package main

import (
	"crypto/rand"
	"math/big"
	mathrand "math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func generateTestURL(t *testing.T) string {
	t.Helper()

	// generate PROTO
	proto := "http://"
	if mathrand.Float32() < 0.5 {
		proto = "https://"
	}

	// generate DOMAIN
	var letters = "0123456789abcdefghijklmnopqrstuvwxyz"

	minLen, maxLen := 5, 15
	domainLen := mathrand.Intn(maxLen-minLen) + minLen

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
