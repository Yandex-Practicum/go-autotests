package main

// Basic imports
import (
	"os"
	"strings"

	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/suite"
)

// Iteration16Suite is a suite of autotests
type Iteration16Suite struct {
	suite.Suite
}

// SetupSuite bootstraps suite dependencies
func (suite *Iteration16Suite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagBaseProfilePath, "-base-profile-path non-empty flag required")
	suite.Require().NotEmpty(flagResultProfilePath, "-result-profile-path non-empty flag required")
	suite.Require().NotEmpty(flagPackageName, "-package-name non-empty flag required")
}

// TestProfilesDiff attempts to detect positive differences between two pprof profiles
func (suite *Iteration16Suite) TestProfilesDiff() {
	baseFd, err := os.Open(flagBaseProfilePath)
	suite.Require().NoError(err, "Невозможно открыть файл с базовым профилем: %s", flagBaseProfilePath)
	defer baseFd.Close()

	resultFd, err := os.Open(flagResultProfilePath)
	suite.Require().NoError(err, "Невозможно открыть файл с результирующим профилем: %s", flagResultProfilePath)
	defer resultFd.Close()

	baseProfile, err := profile.Parse(baseFd)
	suite.Assert().NoError(err, "Невозможно распарсить базовый профиль")

	resultProfile, err := profile.Parse(resultFd)
	suite.Assert().NoError(err, "Невозможно распарсить результирующий профиль")

	baseProfile.Scale(-1)
	mergedProfile, err := profile.Merge([]*profile.Profile{resultProfile, baseProfile})

	// inspect only target package functions samples
	for i, sample := range mergedProfile.Sample {
		if len(mergedProfile.Function) < i {
			break
		}

		fn := mergedProfile.Function[i]
		fName := strings.ToLower(fn.Name)

		// inspect only target package non-test functions
		if !strings.Contains(fName, flagPackageName) ||
			strings.Contains(fName, "test_run") {
			continue
		}

		for _, value := range sample.Value {
			if value < 0 {
				return
			}
		}
	}

	suite.T().Error("Не удалось обнаружить положительных изменений в результирующем профиле")
}
