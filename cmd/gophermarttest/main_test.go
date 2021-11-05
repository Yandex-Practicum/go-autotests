package main

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
