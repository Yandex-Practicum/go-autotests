package main

import (
	"bytes"
	"context"
	"embed"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/lrsk/gopacket/bytediff"
	"github.com/stretchr/testify/suite"
)

type Iteration21Suite struct {
	suite.Suite
}

func (suite *Iteration21Suite) SetupSuite() {
	suite.Require().NotEmpty(flagResetBinaryPath, "-reset-path not-empty flag required")
}

//go:embed testdata/*
var testFiles embed.FS

type testCase struct {
	inputFile    string
	expectedFile string
}

func (suite *Iteration21Suite) TestResetGenerator(t *testing.T) {
	testCases := []testCase{
		{
			inputFile:    "testdata/simple_struct.go",
			expectedFile: "testdata/simple_struct.golden",
		},
		{
			inputFile:    "testdata/complex_struct.go",
			expectedFile: "testdata/complex_struct.golden",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.inputFile, func(t *testing.T) {
			tmpDir := t.TempDir()
			inputPath := filepath.Join(tmpDir, "input.go")
			outputPath := filepath.Join(tmpDir, "reset.gen.go")

			inputData, err := testFiles.ReadFile(tc.inputFile)
			suite.Assert().NoError(err, "Ошибка чтения входного файла")

			expected, err := testFiles.ReadFile(tc.expectedFile)
			suite.Assert().NoError(err, "Ошибка чтения данных из файла golden")

			err = os.WriteFile(inputPath, inputData, 0644)
			suite.Assert().NoError(err, "Ошибка записи данных в файл")

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, flagResetBinaryPath, inputPath)
			var buf bytes.Buffer
			cmd.Stdout = &buf
			cmd.Stderr = &buf
			err = cmd.Run()
			suite.Assert().NoError(err, "Ошибка запуска генератора")

			actual, err := os.ReadFile(outputPath)
			suite.Assert().NoError(err, "Ошибка чтения данных из сгенерируемого файла")

			if string(actual) != string(expected) {
				diffStr := bytediff.Diff(actual, expected)

				t.Errorf("Сгенерированный код не совпадает с ожидаемым:\n\n"+
					"Ожидаемый результат:\n%s\n\n"+
					"Полученный результат:\n%s\n\n"+
					"Дифф:\n%v",
					expected, actual, diffStr)
			}
		})
	}
}
