//go:build !acceptance

package ftracker_test

// Ниже представлены стаб-функции для корректной компиляции кода тестов вне CI окружения

func ShowTrainingInfo(_ int, _ string, _, _, _ float64, _, _ int) string {
	return ""
}

func RunningSpentCalories(_ int, _, _ float64) float64 {
	return 0
}

func WalkingSpentCalories(_ int, _, _, _ float64) float64 {
	return 0
}

func SwimmingSpentCalories(_, _ int, _, _ float64) float64 {
	return 0
}
