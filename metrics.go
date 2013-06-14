package main

import (
	"sync"
	"time"
)

type Measurement struct {
	Key string
	Val uint64
}

type MetricsFunc func() Measurement

type Metrics struct {
	Inbox    chan Measurement
	counters map[string]uint64
	funcs    []MetricsFunc
	fMu      *sync.Mutex
}

func NewMetrics() *Metrics {
	metrics := new(Metrics)
	metrics.Inbox = make(chan Measurement)
	metrics.counters = make(map[string]uint64)
	metrics.funcs = make([]MetricsFunc, 0)
	metrics.fMu = new(sync.Mutex)
	return metrics
}

func (m *Metrics) Start() {
	go m.Run()
}

func (m *Metrics) Run() {
	ticker := time.Tick(1 * time.Second)
	for {
		select {
		case measurement := <-m.Inbox:
			m.counters[measurement.Key] += measurement.Val
		case <-ticker:
			for k := range m.counters {
				Logf("measure.%s=%d", k, m.counters[k])
				m.counters[k] = 0
			}
			for _, f := range m.funcs {
				measurement := f()
				Logf("measure.%s=%d", measurement.Key, measurement.Val)
			}
		}
	}
}

func (m *Metrics) RegisterFunc(f MetricsFunc) {
	m.fMu.Lock()
	defer m.fMu.Unlock()
	m.funcs = append(m.funcs, f)
}

func NewCount(key string, n uint64) Measurement {
	return Measurement{Key: key, Val: n}
}
