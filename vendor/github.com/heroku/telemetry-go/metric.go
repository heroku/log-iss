package telemetry

import (
	"context"
	"fmt"

	metricApi "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
)

// MeterProvider returns the registered global meter provider.
// If none is registered then a No-op MeterProvider is returned.
func MeterProvider() *metric.MeterProvider {
	if config.MeterProvider == nil {
		return metric.NewMeterProvider()
	}
	return config.MeterProvider
}

// Meter provides access to instrument instances for recording metrics.
func Meter() metricApi.Meter {
	if config.Meter == nil {
		return MeterProvider().Meter(MeterName)
	}
	return config.Meter
}

// Int64Count recording increasing values.
func Int64Count(ctx context.Context, name string, value int64) {
	counter, err := Meter().Int64Counter(name)
	if err != nil {
		errorHandling(err)
	}
	counter.Add(ctx, value)
}

// Float64Count recording increasing values.
func Float64Count(ctx context.Context, name string, value float64) {
	counter, err := Meter().Float64Counter(name)
	if err != nil {
		errorHandling(err)
	}
	counter.Add(ctx, value)
}

// Float64UpDownCount recording changes of a value.
func Float64UpDownCount(ctx context.Context, name string, value float64) {
	counter, err := Meter().Float64UpDownCounter(name)
	if err != nil {
		errorHandling(err)
	}
	counter.Add(ctx, value)
}

// Int64UpDownCount recording changes of a value.
func Int64UpDownCount(ctx context.Context, name string, value int64) {
	counter, err := Meter().Int64UpDownCounter(name)
	if err != nil {
		errorHandling(err)
	}
	counter.Add(ctx, value)
}

// Int64Histogram recording a distribution of values.
func Int64Histogram(ctx context.Context, name string, value int64) {
	histogram, err := Meter().Int64Histogram(name)
	if err != nil {
		errorHandling(err)
	}
	histogram.Record(ctx, value)
}

// Float64Histogram recording a distribution of values.
func Float64Histogram(ctx context.Context, name string, value float64) {
	histogram, err := Meter().Float64Histogram(name)
	if err != nil {
		errorHandling(err)
	}
	histogram.Record(ctx, value)
}

func errorHandling(err error) {
	// TODO: define where to log or send it
	_, _ = fmt.Println(err)
}
