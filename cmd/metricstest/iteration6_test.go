package main

import (
	"errors"

	"github.com/stretchr/testify/suite"
)

type Iteration6Suite struct {
	suite.Suite

	knownFrameworks []string
}

func (suite *Iteration6Suite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")

	suite.knownFrameworks = []string{
		"github.com/apex",
		"github.com/go-kit/kit/log",
		"github.com/golang/glog",
		"github.com/grafov/kiwi",
		"github.com/inconshreveable/log15",
		"github.com/mgutz/logxi",
		"github.com/rs/zerolog",
		"github.com/sirupsen/logrus",
		"go.uber.org/zap",
		"gopkg.in/inconshreveable/log15",
		"gopkg.in/inconshreveable/log15.v2",
		"log",
	}
}

func (suite *Iteration6Suite) TestLoggerFrameworkUsage() {
	err := usesKnownPackage(suite.T(), flagTargetSourcePath, suite.knownFrameworks)
	if errors.Is(err, errUsageFound) {
		return
	}
	if err == nil || errors.Is(err, errUsageNotFound) {
		suite.T().Errorf("Не найдено использование хотя бы одного известного фреймворка для работы с логами по пути %s", flagTargetSourcePath)
		return
	}
	suite.T().Errorf("Неожиданная ошибка при поиске использования фреймворка по пути %q, %v", flagTargetSourcePath, err)
}
