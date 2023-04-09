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

func (s *TestServerT) checkReceiveValues(gauges []string, needCount int) {
	s.m.Lock()
	defer s.m.Unlock()

	for _, name := range gauges {
		values := s.gauges[name]
		s.e.Len(values, needCount, "Количество значений, полученных по метрике расходится с ожидаемым")
	}
}
