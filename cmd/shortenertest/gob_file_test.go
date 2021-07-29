package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGobFile checks that student uses persistent gob file
func TestGobFile(t *testing.T) {
	require.FileExists(t, config.GobFilePath)

	fstat, err := os.Stat(config.GobFilePath)
	require.NoError(t, err)

	assert.Falsef(t, fstat.IsDir(), "gob is a directory")
	assert.NotEmpty(t, fstat.Size())
}