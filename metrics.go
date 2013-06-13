package main

import (
	"time"
)

type Measurement struct {
	Key string
	Val uint64
}

type Metrics struct {
	Inbox    chan Measurement
	counters map[string]uint64
}

func NewMetrics() *Metrics {
	metrics := new(Metrics)
	metrics.Inbox = make(chan Measurement)
	metrics.counters = make(map[string]uint64)
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
		}
	}
}

func (m *Metrics) Sum(key string, n uint64) {
	m.Inbox <- Measurement{Key: key, Val: n}
}

func (m *Metrics) Count(key string) {
	m.Sum(key, 1)
}
