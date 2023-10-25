//go:build !acceptance

package ftracker_test

const (
	lenStep   = 0.65
	mInKm     = 1000
	minInH    = 60
	kmhInMsec = 0.278
	cmInM     = 100

	walkingCaloriesWeightMultiplier = 0.035
	walkingSpeedHeightMultiplier    = 0.029

	runningCaloriesMeanSpeedMultiplier = 18
	runningCaloriesMeanSpeedShift      = 1.79

	swimmingLenStep                  = 1.38
	swimmingCaloriesMeanSpeedShift   = 1.1
	swimmingCaloriesWeightMultiplier = 2
)

func meanSpeed(action int, duration float64) float64 {
	if duration <= 0 {
		return 0
	}
	d := distance(action)
	return d / duration
}

func swimmingMeanSpeed(lengthPool, countPool int, duration float64) float64 {
	if duration == 0 {
		return 0
	}
	return float64(lengthPool) * float64(countPool) / mInKm / duration
}

func distance(action int) float64 {
	return float64(action) * lenStep / mInKm
}
