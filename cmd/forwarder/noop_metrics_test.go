package main

import (
	"testing"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
)

var noop = NewNoopRegistry()

func TestNoopRegisrty_GetOrRegister(t *testing.T) {
	assert := assert.New(t)

	counter := metrics.GetOrRegisterCounter("counter", noop)
	timer := metrics.GetOrRegisterTimer("timer", noop)

	var ok bool
	_, ok = counter.(metrics.Counter)
	assert.True(ok)

	_, ok = timer.(metrics.Timer)
	assert.True(ok)
}

func TestNoopRegistry_Inc(t *testing.T) {
	assert := assert.New(t)

	counter := metrics.GetOrRegisterCounter("counter", noop)
	counter.Inc(1)

	// Ensure no panic
	assert.True(true)
}
