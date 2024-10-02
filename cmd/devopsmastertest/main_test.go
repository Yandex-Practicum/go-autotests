package main

//go:generate go test -c -o=../../bin/devopsmastertest

import (
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestLesson01(t *testing.T) {
	suite.Run(t, new(Lesson01Suite))
}

func TestLesson02(t *testing.T) {
	suite.Run(t, new(Lesson02Suite))
}
