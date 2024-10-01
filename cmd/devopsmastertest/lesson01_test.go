package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
	"github.com/stretchr/testify/suite"
)

// Lesson01Suite является сьютом с тестами урока
type Lesson01Suite struct {
	suite.Suite
}

func (suite *Lesson01Suite) TestServerStats() {
	// проверяем наличие необходимых флагов
	suite.Require().NotEmpty(flagTargetBinaryPath, "-binary-path non-empty flag required")

	// генерируем набор сценариев тестирования
	suite.T().Log("generating scenarios")
	respSet := newResponseSet()

	maxRequests := len(respSet)
	var stats []serverStat
	var setOutputs []string
	for _, resp := range respSet {
		stats = append(stats, resp.stats)
		setOutputs = append(setOutputs, resp.expectedOutput...)
	}

	suite.T().Log("creating handler")
	reqNotifier := make(chan int)
	handler := newFaultySrvHandler(stats, reqNotifier)

	// запускаем сервер
	suite.T().Log("staring HTTP server")
	go func() {
		err := http.ListenAndServe("127.0.0.1:80", handler)
		if err != nil {
			suite.FailNowf("cannot start HTTP server", "error: %s", err)
		}
	}()

	// запускаем бинарник скрипта
	suite.T().Log("creating process")
	scriptProc := fork.NewBackgroundProcess(context.Background(), flagTargetBinaryPath)

	binctx, bincancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer bincancel()

	suite.T().Log("starting process")
	if err := scriptProc.Start(binctx); err != nil {
		suite.FailNowf("cannot start script process", "error: %s", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	// ждем завершения
	suite.T().Log("waiting scenarios to complete")
	var requestsMade int
	func() {
		for {
			select {
			case <-sigChan:
				// получен сигнал завершения
				return
			case <-ctx.Done():
				// время вышло
				return
			case requestsMade = <-reqNotifier:
				suite.T().Logf("got request %d", requestsMade)
				if requestsMade == maxRequests {
					// все сценарии были обработаны
					return
				}
			}
		}
	}()

	// останавливаем процесс скрипта
	suite.T().Log("stopping process")
	_, err := scriptProc.Stop(syscall.SIGINT, syscall.SIGKILL)
	if err != nil {
		suite.FailNowf("cannot stop script process", "error: %s", err)
		return
	}

	// сравниваем вывод скрпта в консоль с ожидаемым выводом
	expectedOutput := strings.Join(setOutputs, "\n")
	if expectedOutput != "" {
		expectedOutput += "\n"
	}

	suite.T().Log("checking results")
	stdout := scriptProc.Stdout(context.Background())
	suite.Assert().Equal(expectedOutput, string(stdout), "Вывод скрипта отличается от ожидаемого")
}

func newFaultySrvHandler(stats []serverStat, notifier chan<- int) http.HandlerFunc {
	var mu sync.Mutex
	var receivedRequestsCount int
	return func(w http.ResponseWriter, r *http.Request) {
		// не даем делать запросы в многопоточном режиме, чтобы сохранить консистентность обработки/вывода результатов
		mu.Lock()
		defer mu.Unlock()

		if receivedRequestsCount >= len(stats) {
			// отвечаем ошибкой если у нас кончились заготовленные ответы,
			// а запросы все еще приходят
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		body, err := stats[receivedRequestsCount].MarshalText()
		if err != nil {
			// почему-то не смогли закодировать данные сервера в строку
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// отправляем ответ
		_, _ = w.Write(body)
		// увеличиваем счетчик принятых запросов
		receivedRequestsCount++
		// оповещаем тест о новом обработанном запросе
		notifier <- receivedRequestsCount
	}
}

type responseSet []responsePair

type responsePair struct {
	stats          serverStat
	expectedOutput []string
}

type serverStat struct {
	CurrentLA             int
	MemBytesAvailable     int
	MemBytesUsed          int
	DiskBytesAvailable    int
	DiskBytesUsed         int
	NetBandwidthAvailable int
	NetBandwidthUsed      int
}

func (s serverStat) MarshalText() ([]byte, error) {
	m := fmt.Sprintf("%d,%d,%d,%d,%d,%d,%d",
		s.CurrentLA,
		s.MemBytesAvailable,
		s.MemBytesUsed,
		s.DiskBytesAvailable,
		s.DiskBytesUsed,
		s.NetBandwidthAvailable,
		s.NetBandwidthUsed,
	)
	return []byte(m), nil
}

const (
	unitB  = 1
	unitKb = unitB * 1024
	unitMb = unitKb * 1024
	unitGb = unitMb * 1024

	unitBps  = 1.0
	unitKbps = unitBps * 1000
	unitMbps = unitKbps * 1000
	unitGbps = unitMbps * 1000
)

func newResponseSet() (res responseSet) {
	src := rand.NewSource(time.Now().UnixNano())
	rnd := rand.New(src)

	// изначальная конфигурация сервера
	memBytesAvailable := intInRange(rnd, 4*unitGb, 5*unitGb)
	diskBytesAvailable := intInRange(rnd, 256*unitGb, 512*unitGb)
	netBandwidthAvailable := intInRange(rnd, 1*unitGbps, 10*unitGbps)

	{
		// сценарий: все в порядке
		res = append(res, responsePair{
			stats: serverStat{
				CurrentLA:             intInRange(rnd, 0, 29),
				MemBytesAvailable:     memBytesAvailable,
				MemBytesUsed:          intInRange(rnd, memBytesAvailable/3, memBytesAvailable/2),
				DiskBytesAvailable:    diskBytesAvailable,
				DiskBytesUsed:         intInRange(rnd, diskBytesAvailable/5, diskBytesAvailable/3),
				NetBandwidthAvailable: netBandwidthAvailable,
				NetBandwidthUsed:      intInRange(rnd, netBandwidthAvailable/8, netBandwidthAvailable/4),
			},
			expectedOutput: nil,
		})
	}

	{
		// сценарий: слишком большое LA
		currentLA := intInRange(rnd, 30, 99)
		res = append(res, responsePair{
			stats: serverStat{
				CurrentLA:             currentLA,
				MemBytesAvailable:     memBytesAvailable,
				MemBytesUsed:          intInRange(rnd, memBytesAvailable/3, memBytesAvailable/2),
				DiskBytesAvailable:    diskBytesAvailable,
				DiskBytesUsed:         intInRange(rnd, diskBytesAvailable/5, diskBytesAvailable/3),
				NetBandwidthAvailable: netBandwidthAvailable,
				NetBandwidthUsed:      intInRange(rnd, netBandwidthAvailable/8, netBandwidthAvailable/4),
			},
			expectedOutput: []string{
				fmt.Sprintf("Load Average is too high: %d", currentLA),
			},
		})
	}

	// встряхиваем набор сценариев
	rnd.Shuffle(len(res), func(i, j int) {
		res[i], res[j] = res[j], res[i]
	})
	return
}

func intInRange(rnd *rand.Rand, min, max int) int {
	return rnd.Intn(max-min) + min
}
