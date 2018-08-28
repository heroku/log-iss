package main

import (
	"reflect"

	metrics "github.com/rcrowley/go-metrics"
)

type NoopRegistry struct{}

func NewNoopRegistry() metrics.Registry {
	noop := NoopRegistry{}
	return noop
}

func (n NoopRegistry) Each(_ func(string, interface{})) {}

func (n NoopRegistry) Get(_ string) (r interface{}) {
	return
}

func (n NoopRegistry) GetAll() (r map[string]map[string]interface{}) {
	return
}

func (n NoopRegistry) GetOrRegister(_ string, i interface{}) interface{} {
	if v := reflect.ValueOf(i); v.Kind() == reflect.Func {
		i = v.Call(nil)[0].Interface()
	}

	return i
}

func (n NoopRegistry) Register(_ string, _ interface{}) (r error) {
	return
}

func (n NoopRegistry) RunHealthchecks() {}

func (n NoopRegistry) Unregister(string) {}

func (n NoopRegistry) UnregisterAll() {}
