package telemetry

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/trace"
)

// TestExporter exporter for collecting spans in memory for testing
type TestExporter struct {
	mu      sync.Mutex
	spans   []trace.ReadOnlySpan
	metrics *metricdata.ResourceMetrics
}

func (t *TestExporter) Temporality(kind metric.InstrumentKind) metricdata.Temporality {
	return metricdata.DeltaTemporality
}

func (t *TestExporter) Aggregation(kind metric.InstrumentKind) metric.Aggregation {
	return metric.DefaultAggregationSelector(kind)
}

func (t *TestExporter) ForceFlush(ctx context.Context) error {
	return nil
}

func (t *TestExporter) ExportSpans(ctx context.Context, s []trace.ReadOnlySpan) error {
	t.mu.Lock()
	t.spans = append(t.spans, s...)
	t.mu.Unlock()

	return nil
}

func (t *TestExporter) GetRecordedMainSpans() []trace.ReadOnlySpan {
	filtered := []trace.ReadOnlySpan{}

	for _, span := range t.spans {
		for _, kv := range span.Attributes() {
			if kv.Key == logExporterMainKey {
				if kv.Value.AsBool() {
					filtered = append(filtered, span)
				}
			}
		}
	}

	return filtered
}

func (t *TestExporter) GetRecordedSpans() []trace.ReadOnlySpan {
	return t.spans
}

func (t *TestExporter) ClearRecordedSpans() {
	t.spans = []trace.ReadOnlySpan{}
}

func (t *TestExporter) Export(ctx context.Context, resourceMetrics *metricdata.ResourceMetrics) error {
	t.mu.Lock()
	t.metrics = resourceMetrics
	t.mu.Unlock()

	return nil
}

func (t *TestExporter) GetRecordedMetrics() *metricdata.ResourceMetrics {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.metrics
}

func (t *TestExporter) Shutdown(ctx context.Context) error {
	return nil
}
