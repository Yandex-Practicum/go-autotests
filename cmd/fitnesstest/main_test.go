package ftracker

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

func TestFitnessSuite(t *testing.T) {
	suite.Run(t, new(FitnessSuite))
}

type FitnessSuite struct {
	suite.Suite
}

func (s *FitnessSuite) TestShowTrainingInfo() {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	actionsNum := int(rnd.Int63n(10000-1000) + 1000)
	durationNum := float64(rnd.Int63n(3)) + rnd.Float64()
	weightNum := float64(rnd.Int63n(140-80) + 80)
	heightNum := float64(rnd.Int63n(220-150) + 150)
	lengthPoolNum := int(rnd.Int63n(50-10) + 10)
	countPoolNum := int(rnd.Int63n(10-1) + 1)

	s.Run("rinning", func() {
		trainingType := "Бег"
		res := ShowTrainingInfo(actionsNum, trainingType, durationNum, weightNum, heightNum, lengthPoolNum, countPoolNum)

		distance := testDistance(actionsNum)
		speed := testMeanSpeed(actionsNum, durationNum)
		calories := RunningSpentCalories(actionsNum, weightNum, durationNum)
		expected := fmt.Sprintf("Тип тренировки: %s\nДлительность: %.2f ч.\nДистанция: %.2f км.\nСкорость: %.2f км/ч\nСожгли калорий: %.2f\n", trainingType, durationNum, distance, speed, calories)

		s.Assert().Equal(expected, res, "Результат выполнения функции ShowTrainingInfo не совпадает с ожидаемым")
	})

	s.Run("walking", func() {
		trainingType := "Ходьба"
		res := ShowTrainingInfo(actionsNum, trainingType, durationNum, weightNum, heightNum, lengthPoolNum, countPoolNum)

		distance := testDistance(actionsNum)
		speed := testMeanSpeed(actionsNum, durationNum)
		calories := WalkingSpentCalories(actionsNum, durationNum, weightNum, heightNum)
		expected := fmt.Sprintf("Тип тренировки: %s\nДлительность: %.2f ч.\nДистанция: %.2f км.\nСкорость: %.2f км/ч\nСожгли калорий: %.2f\n", trainingType, durationNum, distance, speed, calories)

		s.Assert().Equal(expected, res, "Результат выполнения функции ShowTrainingInfo не совпадает с ожидаемым")
	})

	s.Run("swimming", func() {
		trainingType := "Плавание"
		res := ShowTrainingInfo(actionsNum, trainingType, durationNum, weightNum, heightNum, lengthPoolNum, countPoolNum)

		distance := testDistance(actionsNum)
		speed := testSwimmingMeanSpeed(lengthPoolNum, countPoolNum, durationNum)
		calories := SwimmingSpentCalories(lengthPoolNum, countPoolNum, durationNum, weightNum)
		expected := fmt.Sprintf("Тип тренировки: %s\nДлительность: %.2f ч.\nДистанция: %.2f км.\nСкорость: %.2f км/ч\nСожгли калорий: %.2f\n", trainingType, durationNum, distance, speed, calories)

		s.Assert().Equal(expected, res, "Результат выполнения функции ShowTrainingInfo не совпадает с ожидаемым")
	})

	s.Run("unknown", func() {
		actionsNum := int(rnd.Int63n(10000-1000) + 1000)
		trainingType := randString(3, 15)
		durationNum := float64(rnd.Int63n(3)) + rnd.Float64()
		weightNum := float64(rnd.Int63n(140-80) + 80)
		heightNum := float64(rnd.Int63n(220-150) + 150)
		lengthPoolNum := int(rnd.Int63n(50-10) + 10)
		countPoolNum := int(rnd.Int63n(10-1) + 1)

		res := ShowTrainingInfo(actionsNum, trainingType, durationNum, weightNum, heightNum, lengthPoolNum, countPoolNum)
		s.Assert().Equal("неизвестный тип тренировки", res)
	})
}

func (s *FitnessSuite) TestWalkingSpentCalories() {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	actionsNum := int(rnd.Int63n(10000-1000) + 1000)
	durationNum := float64(rnd.Int63n(3)) + rnd.Float64()
	weightNum := float64(rnd.Int63n(140-80) + 80)
	heightNum := float64(rnd.Int63n(220-150) + 150)

	meanSpeed := testMeanSpeed(actionsNum, durationNum)
	expected := (_walkingCaloriesWeightMultiplier*weightNum + (math.Pow(meanSpeed*_kmhInMsec, 2.0)/(heightNum/_cmInM))*_walkingSpeedHeightMultiplier*weightNum) * durationNum * _minInH

	res := WalkingSpentCalories(actionsNum, durationNum, weightNum, heightNum)
	s.Assert().InDelta(expected, res, 0.05, "Значение полученное из функции WalkingSpentCalories не совпадает с ожидаемым")
}

func (s *FitnessSuite) TestRunningSpentCalories() {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	actionsNum := int(rnd.Int63n(10000-1000) + 1000)
	durationNum := float64(rnd.Int63n(3)) + rnd.Float64()
	weightNum := float64(rnd.Int63n(140-80) + 80)

	meanSpeed := testMeanSpeed(actionsNum, durationNum)
	expected := ((_runningCaloriesMeanSpeedMultiplier * meanSpeed * _runningCaloriesMeanSpeedShift) * weightNum / _mInKm * durationNum * _minInH)

	res := RunningSpentCalories(actionsNum, weightNum, durationNum)
	s.Assert().InDelta(expected, res, 0.05, "Значение полученное из функции RunningSpentCalories не совпадает с ожидаемым")
}

func (s *FitnessSuite) TestSwimmingSpentCalories() {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	lengthPoolNum := int(rnd.Int63n(50-10) + 10)
	countPoolNum := int(rnd.Int63n(10-1) + 1)
	durationNum := float64(rnd.Int63n(3)) + rnd.Float64()
	weightNum := float64(rnd.Int63n(140-80) + 80)

	meanSpeed := testSwimmingMeanSpeed(lengthPoolNum, countPoolNum, durationNum)
	expected := (meanSpeed + _swimmingCaloriesMeanSpeedShift) * _swimmingCaloriesWeightMultiplier * weightNum * durationNum

	res := SwimmingSpentCalories(lengthPoolNum, countPoolNum, durationNum, weightNum)
	s.Assert().InDelta(expected, res, 0.05, "Значение полученное из функции SwimmingSpentCalories не совпадает с ожидаемым")
}

func randString(minLen, maxLen int) string {
	var letters = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFJHIJKLMNOPQRSTUVWXYZ"

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	slen := rnd.Intn(maxLen-minLen) + minLen

	s := make([]byte, 0, slen)
	i := 0
	for len(s) < slen {
		idx := rnd.Intn(len(letters) - 1)
		char := letters[idx]
		if i == 0 && '0' <= char && char <= '9' {
			continue
		}
		s = append(s, char)
		i++
	}

	return string(s)
}

const (
	_lenStep   = 0.65
	_mInKm     = 1000
	_minInH    = 60
	_kmhInMsec = 0.278
	_cmInM     = 100

	_walkingCaloriesWeightMultiplier = 0.035
	_walkingSpeedHeightMultiplier    = 0.029

	_runningCaloriesMeanSpeedMultiplier = 18
	_runningCaloriesMeanSpeedShift      = 1.79

	_swimmingLenStep                  = 1.38
	_swimmingCaloriesMeanSpeedShift   = 1.1
	_swimmingCaloriesWeightMultiplier = 2
)

func testMeanSpeed(action int, duration float64) float64 {
	if duration <= 0 {
		return 0
	}
	d := testDistance(action)
	return d / duration
}

func testSwimmingMeanSpeed(lengthPool, countPool int, duration float64) float64 {
	if duration == 0 {
		return 0
	}
	return float64(lengthPool) * float64(countPool) / _mInKm / duration
}

func testDistance(action int) float64 {
	return float64(action) * _lenStep / _mInKm
}
