package main

import (
	"errors"

	"github.com/stretchr/testify/suite"
)

type Iteration6Suite struct {
	suite.Suite

	knownLoggers PackageRules
}

// SetupSuite подготавливает необходимые зависимости
func (suite *Iteration6Suite) SetupSuite() {
	// проверяем наличие необходимых флагов
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")

	// список известных логгеров
	suite.knownLoggers = PackageRules{
		{Name: "github.com/rs/zerolog"},
		{Name: "go.uber.org/zap"},
		{Name: "github.com/sirupsen/logrus"},
		{Name: "log/slog"},
		// "github.com/apex",
		// "github.com/go-kit/kit/log",
		// "github.com/golang/glog",
		// "github.com/grafov/kiwi",
		// "github.com/inconshreveable/log15",
		// "github.com/mgutz/logxi",
		// "gopkg.in/inconshreveable/log15",
		// "gopkg.in/inconshreveable/log15.v2",
		// "log",
	}
}

// TestLoggerUsage пробует рекурсивно найти хотя бы одно использование известных логгеров в директории с исходным кодом проекта
func (suite *Iteration6Suite) TestLoggerUsage() {
	// проверяем наличие известных фреймворков
	err := usesKnownPackage(suite.T(), flagTargetSourcePath, suite.knownLoggers...)
	if errors.Is(err, errUsageFound) {
		return
	}
	if errors.Is(err, errUsageNotFound) {
		suite.T().Errorf("Не найдено использование хотя бы одного известного логгера по пути %s", flagTargetSourcePath)
		return
	}
	suite.T().Errorf("Неожиданная ошибка при поиске использования логгера по пути %s: %s", flagTargetSourcePath, err)
}
