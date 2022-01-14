package main

//go:generate go test -c -o=../../bin/gophermarttest

import (
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestGophermart(t *testing.T) {
	suite.Run(t, new(GophermartSuite))
}

func TestAccrual(t *testing.T) {
	suite.Run(t, new(AccrualSuite))
}
