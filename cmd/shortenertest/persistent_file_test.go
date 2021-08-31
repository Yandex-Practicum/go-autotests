package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPersistentFile checks that student uses persistent gob file
func TestPersistentFile(t *testing.T) {
	require.FileExists(t, config.PersistentFilePath)

	fstat, err := os.Stat(config.PersistentFilePath)
	require.NoError(t, err)

	assert.Falsef(t, fstat.IsDir(), "file is a directory")
	assert.NotEmpty(t, fstat.Size())
}

// for backward compatibility
func TestGobFile(t *testing.T) {
	TestPersistentFile(t)
}
