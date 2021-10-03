package random

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDomain(t *testing.T) {
	generated := make(map[string]struct{})

	for i := 0; i < 5000; i++ {
		domain := Domain(5, 15)
		require.NotContains(t, generated, domain)
		generated[domain] = struct{}{}
	}
}

func TestURL(t *testing.T) {
	generated := make(map[string]struct{})

	for i := 0; i < 5000; i++ {
		u := URL().String()
		require.NotContains(t, generated, u)
		generated[u] = struct{}{}
	}
}