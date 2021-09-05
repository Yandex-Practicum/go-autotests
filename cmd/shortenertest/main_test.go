package main

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

func TestIteration2(t *testing.T) {
	suite.Run(t, new(Iteration2Suite))
}

func TestIteration3(t *testing.T) {
	suite.Run(t, new(Iteration3Suite))
}

func TestIteration4(t *testing.T) {
	suite.Run(t, new(Iteration4Suite))
}

func TestIteration5(t *testing.T) {
	suite.Run(t, new(Iteration5Suite))
}
