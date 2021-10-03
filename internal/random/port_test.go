package random

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPort(t *testing.T) {
	generated := make(map[int]struct{})

	for i := 0; i < 100; i++ {
		port := Port(1024, 65535)
		require.NotContains(t, generated, port)
		generated[port] = struct{}{}
	}
}
