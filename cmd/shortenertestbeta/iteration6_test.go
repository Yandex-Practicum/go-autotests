package main

import (
	"errors"

	"github.com/stretchr/testify/suite"
)

// Iteration9Suite является сьютом с тестами и состоянием для инкремента
type Iteration6Suite struct {
	suite.Suite

	knownLoggers []string
}

// SetupSuite подготавливает необходимые зависимости
func (suite *Iteration6Suite) SetupSuite() {
	// проверяем наличие необходимых флагов
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")

	// список известных логгеров
	suite.knownLoggers = []string{
		"github.com/rs/zerolog",
		"go.uber.org/zap",
		"github.com/sirupsen/logrus",
	}
}

// TestLoggerUsage пробует рекурсивно найти хотя бы одно использование известных логгеров в директории с исходным кодом проекта
func (suite *Iteration6Suite) TestLoggerUsage() {
	// проверяем наличие известных фреймворков
	err := usesKnownPackage(suite.T(), flagTargetSourcePath, suite.knownLoggers)
	if err == nil {
		return
	}
	if errors.Is(err, errUsageNotFound) {
		suite.T().Errorf("Не найдено использование хотя бы одного известного логгера по пути %s", flagTargetSourcePath)
		return
	}
	suite.T().Errorf("Неожиданная ошибка при поиске использования логгера по пути %s: %s", flagTargetSourcePath, err)
}
