package ftracker_test

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/Yandex-Practicum/go-autotests/internal/random"
	"github.com/stretchr/testify/assert"
)

func TestShowTrainingInfo(t *testing.T) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	actionsNum := int(rnd.Int63n(10000-1000) + 1000)
	durationNum := float64(rnd.Int63n(3)) + rnd.Float64()
	weightNum := float64(rnd.Int63n(140-80) + 80)
	heightNum := float64(rnd.Int63n(220-150) + 150)
	lengthPoolNum := int(rnd.Int63n(50-10) + 10)
	countPoolNum := int(rnd.Int63n(10-1) + 1)

	t.Run("rinning", func(t *testing.T) {
		trainingType := "Бег"
		res := ShowTrainingInfo(actionsNum, trainingType, durationNum, weightNum, heightNum, lengthPoolNum, countPoolNum)

		distance := distance(actionsNum)
		speed := meanSpeed(actionsNum, durationNum)
		calories := RunningSpentCalories(actionsNum, weightNum, durationNum)
		expected := fmt.Sprintf("Тип тренировки: %s\nДлительность: %.2f ч.\nДистанция: %.2f км.\nСкорость: %.2f км/ч\nСожгли калорий: %.2f\n", trainingType, durationNum, distance, speed, calories)

		assert.Equal(t, expected, res, "Результат выполнения функции ShowTrainingInfo не совпадает с ожидаемым")
	})

	t.Run("walking", func(t *testing.T) {
		trainingType := "Ходьба"
		res := ShowTrainingInfo(actionsNum, trainingType, durationNum, weightNum, heightNum, lengthPoolNum, countPoolNum)

		distance := distance(actionsNum)
		speed := meanSpeed(actionsNum, durationNum)
		calories := WalkingSpentCalories(actionsNum, durationNum, weightNum, heightNum)
		expected := fmt.Sprintf("Тип тренировки: %s\nДлительность: %.2f ч.\nДистанция: %.2f км.\nСкорость: %.2f км/ч\nСожгли калорий: %.2f\n", trainingType, durationNum, distance, speed, calories)

		assert.Equal(t, expected, res, "Результат выполнения функции ShowTrainingInfo не совпадает с ожидаемым")
	})

	t.Run("swimming", func(t *testing.T) {
		trainingType := "Плавание"
		res := ShowTrainingInfo(actionsNum, trainingType, durationNum, weightNum, heightNum, lengthPoolNum, countPoolNum)

		distance := distance(actionsNum)
		speed := swimmingMeanSpeed(lengthPoolNum, countPoolNum, durationNum)
		calories := SwimmingSpentCalories(lengthPoolNum, countPoolNum, durationNum, weightNum)
		expected := fmt.Sprintf("Тип тренировки: %s\nДлительность: %.2f ч.\nДистанция: %.2f км.\nСкорость: %.2f км/ч\nСожгли калорий: %.2f\n", trainingType, durationNum, distance, speed, calories)

		assert.Equal(t, expected, res, "Результат выполнения функции ShowTrainingInfo не совпадает с ожидаемым")
	})

	t.Run("unknown", func(t *testing.T) {
		actionsNum := int(rnd.Int63n(10000-1000) + 1000)
		trainingType := random.ASCIIString(3, 15)
		durationNum := float64(rnd.Int63n(3)) + rnd.Float64()
		weightNum := float64(rnd.Int63n(140-80) + 80)
		heightNum := float64(rnd.Int63n(220-150) + 150)
		lengthPoolNum := int(rnd.Int63n(50-10) + 10)
		countPoolNum := int(rnd.Int63n(10-1) + 1)

		res := ShowTrainingInfo(actionsNum, trainingType, durationNum, weightNum, heightNum, lengthPoolNum, countPoolNum)
		assert.Equal(t, "неизвестный тип тренировки", res)
	})
}

func TestWalkingSpentCalories(t *testing.T) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	actionsNum := int(rnd.Int63n(10000-1000) + 1000)
	durationNum := float64(rnd.Int63n(3)) + rnd.Float64()
	weightNum := float64(rnd.Int63n(140-80) + 80)
	heightNum := float64(rnd.Int63n(220-150) + 150)

	meanSpeed := meanSpeed(actionsNum, durationNum)
	expected := (walkingCaloriesWeightMultiplier*weightNum + (math.Pow(meanSpeed*kmhInMsec, 2.0)/(heightNum/cmInM))*walkingSpeedHeightMultiplier*weightNum) * durationNum * minInH

	res := WalkingSpentCalories(actionsNum, durationNum, weightNum, heightNum)
	assert.InDelta(t, expected, res, 0.05, "Значение полученное из функции WalkingSpentCalories не совпадает с ожидаемым")
}

func TestRunningSpentCalories(t *testing.T) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	actionsNum := int(rnd.Int63n(10000-1000) + 1000)
	durationNum := float64(rnd.Int63n(3)) + rnd.Float64()
	weightNum := float64(rnd.Int63n(140-80) + 80)

	meanSpeed := meanSpeed(actionsNum, durationNum)
	expected := ((runningCaloriesMeanSpeedMultiplier * meanSpeed * runningCaloriesMeanSpeedShift) * weightNum / mInKm * durationNum * minInH)

	res := RunningSpentCalories(actionsNum, weightNum, durationNum)
	assert.InDelta(t, expected, res, 0.05, "Значение полученное из функции RunningSpentCalories не совпадает с ожидаемым")
}

func TestSwimmingSpentCalories(t *testing.T) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	lengthPoolNum := int(rnd.Int63n(50-10) + 10)
	countPoolNum := int(rnd.Int63n(10-1) + 1)
	durationNum := float64(rnd.Int63n(3)) + rnd.Float64()
	weightNum := float64(rnd.Int63n(140-80) + 80)

	meanSpeed := swimmingMeanSpeed(lengthPoolNum, countPoolNum, durationNum)
	expected := (meanSpeed + swimmingCaloriesMeanSpeedShift) * swimmingCaloriesWeightMultiplier * weightNum * durationNum

	res := SwimmingSpentCalories(lengthPoolNum, countPoolNum, durationNum, weightNum)
	assert.InDelta(t, expected, res, 0.05, "Значение полученное из функции SwimmingSpentCalories не совпадает с ожидаемым")
}
