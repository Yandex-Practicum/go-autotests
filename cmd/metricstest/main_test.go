package main

//go:generate go test -c -o=../../bin/metricstest

import (
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestIteration1(t *testing.T) {
	suite.Run(t, new(Iteration1Suite))
}

func TestIteration2A(t *testing.T) {
	suite.Run(t, new(Iteration2ASuite))
}

func TestIteration2B(t *testing.T) {
	suite.Run(t, new(Iteration2BSuite))
}

func TestIteration3A(t *testing.T) {
	suite.Run(t, new(Iteration3ASuite))
}

func TestIteration3B(t *testing.T) {
	suite.Run(t, new(Iteration3BSuite))
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
