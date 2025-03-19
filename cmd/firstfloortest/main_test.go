package main

//go:generate go test -c -o=../../bin/firstfloortest

import (
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestSprint6Final(t *testing.T) {
	suite.Run(t, new(Sprint6FinalSuite))
}
