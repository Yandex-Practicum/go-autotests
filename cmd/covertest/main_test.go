package covertest

//go:generate go test -c -o=../../bin/covertest

import (
	"errors"
	"fmt"
	"testing"

	"github.com/eltorocorp/drygopher/drygopher/coverage"
	"github.com/eltorocorp/drygopher/drygopher/coverage/analysis"
	"github.com/eltorocorp/drygopher/drygopher/coverage/analysis/raw"
	"github.com/eltorocorp/drygopher/drygopher/coverage/coverageerrors"
	"github.com/eltorocorp/drygopher/drygopher/coverage/host"
	"github.com/eltorocorp/drygopher/drygopher/coverage/packages"
	"github.com/eltorocorp/drygopher/drygopher/coverage/profile"
	"github.com/eltorocorp/drygopher/drygopher/coverage/report"
	"github.com/stretchr/testify/assert"
)

func TestCoverage40(t *testing.T) {
	assert.NoError(t, requireCoverage(40))
}

func TestCoverage55(t *testing.T) {
	assert.NoError(t, requireCoverage(55))
}

func TestCoverage70(t *testing.T) {
	assert.NoError(t, requireCoverage(70))
}

func TestCoverage80(t *testing.T) {
	assert.NoError(t, requireCoverage(80))
}

func requireCoverage(coverage float64) error {
	err := checkCoveragePercentage(coverage)
	if err == nil {
		return nil
	}

	var coverErr coverageerrors.CoverageBelowStandard
	if errors.As(err, &coverErr) {
		return fmt.Errorf("Покрытие тестами ниже требуемого: %w", err)
	}

	var unitErr coverageerrors.UnitTestFailed
	if errors.As(err, &unitErr) {
		return errors.New("Один или несколько юнит-тестов завершились с ошибкой. Проверьте успешность прохождения тестов с флагом `-race`")
	}

	return fmt.Errorf("Невозможно определить степень покрытия кода тестами: %w", err)
}

func checkCoveragePercentage(coverageStandard float64) error {
	suppressProfile := true
	suppressPercentageFile := true
	profileName := ""
	packageExclusions := []string{"/vendor/", "_test"}

	execAPI := new(host.Exec)
	osioAPI := new(host.OSIO)
	packageAPI := packages.New(execAPI, osioAPI)
	profileAPI := profile.New(packageAPI, osioAPI)
	reportAPI := report.New(execAPI)
	rawAPI := raw.New(osioAPI, execAPI)
	analysisAPI := analysis.New(rawAPI)
	coverageAPI := coverage.New(packageAPI, analysisAPI, profileAPI, reportAPI)

	return coverageAPI.AnalyzeUnitTestCoverage(packageExclusions, coverageStandard, suppressProfile, profileName, suppressPercentageFile)

}
