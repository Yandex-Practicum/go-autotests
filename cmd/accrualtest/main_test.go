package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestAccrual(t *testing.T) {
	suite.Run(t, new(AccrualSuite))
}
