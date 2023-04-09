package main

//go:generate go test -c -o=../../bin/metricstest

import (
	"flag"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rekby/fixenv"
	"github.com/stretchr/testify/suite"
)

func TestMain(m *testing.M) {
	flag.Parse()

	gin.SetMode(gin.ReleaseMode)

	fixenv.CreateMainTestEnv(nil)
	os.Exit(m.Run())
}

func TestIteration4(t *testing.T) {
	suite.Run(t, new(Iteration4Suite))
}

func TestIteration5(t *testing.T) {
	suite.Run(t, new(Iteration5Suite))
}

func TestIteration6(t *testing.T) {
	suite.Run(t, new(Iteration6Suite))
}

func TestIteration7(t *testing.T) {
	suite.Run(t, new(Iteration7Suite))
}

func TestIteration8(t *testing.T) {
	suite.Run(t, new(Iteration8Suite))
}

func TestIteration9(t *testing.T) {
	suite.Run(t, new(Iteration9Suite))
}

func TestIteration10A(t *testing.T) {
	suite.Run(t, new(Iteration10ASuite))
}

func TestIteration10B(t *testing.T) {
	suite.Run(t, new(Iteration10BSuite))
}

func TestIteration11(t *testing.T) {
	suite.Run(t, new(Iteration11Suite))
}

func TestIteration12(t *testing.T) {
	suite.Run(t, new(Iteration12Suite))
}

func TestIteration13(t *testing.T) {
	suite.Run(t, new(Iteration13Suite))
}

func TestIteration14(t *testing.T) {
	suite.Run(t, new(Iteration14Suite))
}
