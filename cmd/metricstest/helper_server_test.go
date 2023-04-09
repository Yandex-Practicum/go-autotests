package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
)

type TestServerT struct {
	e        *Env
	endpoint string

	httpServer http.Server
	gin        *gin.Engine

	m        sync.Mutex
	counters map[string][]int64
	gauges   map[string][]float64
}

func NewTestServerT(e *Env, endpoint string) *TestServerT {
	s := &TestServerT{
		e:        e,
		endpoint: endpoint,
		counters: map[string][]int64{},
		gauges:   map[string][]float64{},
		gin:      gin.New(),
	}

	s.httpServer.Addr = s.endpoint
	s.httpServer.Handler = s.gin

	s.gin.RedirectTrailingSlash = false
	s.gin.RedirectFixedPath = false

	s.gin.POST("/update/counter/:name/:value", s.storeCounter)
	s.gin.POST("/update/gauge/:name/:value", s.storeGauge)

	s.gin.Any("/", s.unexpectedCall)
	return s
}

func (s *TestServerT) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *TestServerT) Stop() {
	_ = s.httpServer.Shutdown(context.Background())
}

func (s *TestServerT) storeCounter(c *gin.Context) {
	name := c.Param("name")
	valS := c.Param("value")
	val, err := strconv.ParseInt(valS, 10, 64)
	if err != nil {
		s.e.Errorf("Пришло значение counter не соответствующее формату. name: %q, value: %q, err: %+v", name, valS, err)
		_ = c.AbortWithError(http.StatusBadRequest, fmt.Errorf("parse int value error: %w", err))
		return
	}

	s.m.Lock()
	defer s.m.Unlock()
	s.counters[name] = append(s.counters[name], val)
	s.e.Logf("Получено значение counter %q: %v", name, val)
}

func (s *TestServerT) storeGauge(c *gin.Context) {
	name := c.Param("name")
	valS := c.Param("value")

	val, err := strconv.ParseFloat(valS, 64)
	if err != nil {
		s.e.Errorf("Пришло значение gauge не соответствующее формату. name: %q, value: %q, err: %+v", name, valS, err)
		_ = c.AbortWithError(http.StatusBadRequest, fmt.Errorf("parse float value error: %w", err))
		return
	}

	s.m.Lock()
	defer s.m.Unlock()
	s.gauges[name] = append(s.gauges[name], val)
	s.e.Logf("Получено значение gauge %q: %f", name, val)
}

func (s *TestServerT) unexpectedCall(c *gin.Context) {
	c.AbortWithStatus(http.StatusBadRequest)

	s.e.Errorf("Пришёл неожиданный запрос, method: %q, uri: %q", c.Request.Method, c.Request.RequestURI)
	content, err := io.ReadAll(c.Request.Body)
	if err != nil {
		s.e.Errorf("Не могу прочитать тело запроса:")
		return
	}
	_ = c.Request.Body.Close()
	s.e.Errorf("Тело запроса:\n%s", content)
}

func (s *TestServerT) CheckReceiveValues(gauges []string, counters []string, min, max int) {
	s.m.Lock()
	defer s.m.Unlock()

	for _, name := range gauges {
		values := s.gauges[name]
		lenValues := len(values)
		s.e.True(min <= lenValues && lenValues <= max, "Ожидается количество gauge значений от %v до %v, сейас получено: %v", min, max, lenValues)
	}

	for _, name := range counters {
		values := s.counters[name]
		lenValues := len(values)
		s.e.True(min <= lenValues && lenValues <= max, "Ожидается количество counter значений от %v до %v, сейас получено: %v", min, max, lenValues)
	}
}

func (s *TestServerT) GetLastCounter(name string) int64 {
	s.m.Lock()
	defer s.m.Unlock()

	val := s.counters[name]
	if len(val) > 0 {
		return val[len(val)-1]
	}

	s.e.Errorf("Запрошено значение отсутствующего счётчика counter: %q", name)
	return 0
}

func (s *TestServerT) GetLastGauge(name string) float64 {
	s.m.Lock()
	defer s.m.Unlock()

	val := s.gauges[name]
	if len(val) > 0 {
		return val[len(val)-1]
	}

	s.e.Errorf("Запрошено значение отсутствующего счётчика counter: %q", name)
	return 0
}
