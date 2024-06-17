package telemetry

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

type readOnlySpanWithOverridingResource struct {
	// Embed a ReadOnlySpan so readOnlySpanWithOverridingResource can inherit all the attributes and methods
	sdktrace.ReadOnlySpan
	resource *resource.Resource
}

// Define our own Resource method that will overwrite the method inherited from the embedded ReadOnlySpan
func (s readOnlySpanWithOverridingResource) Resource() *resource.Resource {
	return s.resource
}

type overwriteServiceNameSpanProcessor struct {
	exporter    sdktrace.SpanExporter
	serviceName string
}

func newOverwriteServiceNameSpanProcessor(exporter sdktrace.SpanExporter, serviceName string) *overwriteServiceNameSpanProcessor {
	return &overwriteServiceNameSpanProcessor{exporter: exporter, serviceName: serviceName}
}

func (csp *overwriteServiceNameSpanProcessor) OnStart(parent context.Context, span sdktrace.ReadWriteSpan) {
	// Nothing needs to be done here
}

func (csp *overwriteServiceNameSpanProcessor) OnEnd(span sdktrace.ReadOnlySpan) {
	// Copy span resources and add overwrite service.name
	originalResource := span.Resource()
	modifiedResource, err := resource.Merge(originalResource, resource.NewSchemaless(
		semconv.ServiceNameKey.String(csp.serviceName),
	))
	if err != nil {
		fmt.Println("telemetry.go: ", err)
		return
	}

	// Create a modified copy of the ReadOnlySpan
	modifiedSpan := readOnlySpanWithOverridingResource{
		ReadOnlySpan: span,
		resource:     modifiedResource,
	}

	// Export the modified span
	_ = csp.exporter.ExportSpans(context.Background(), []sdktrace.ReadOnlySpan{modifiedSpan})
}

func (csp *overwriteServiceNameSpanProcessor) Shutdown(ctx context.Context) error {
	return csp.exporter.Shutdown(ctx)
}

func (csp *overwriteServiceNameSpanProcessor) ForceFlush(ctx context.Context) error {
	return csp.exporter.ExportSpans(ctx, []sdktrace.ReadOnlySpan{})
}

func determineKeyType(apiKey string) string {
	if len(apiKey) == 32 || strings.HasPrefix(apiKey, "hcaic_") {
		return "CLASSIC"
	} else {
		return "E_AND_S"
	}
}

func keysAreValid(apiKeys string) bool {
	keys := strings.Split(apiKeys, ";")

	if len(keys) == 0 || len(keys) > 2 {
		return false
	}

	if len(keys) == 2 {
		key1 := determineKeyType(keys[0])
		key2 := determineKeyType(keys[1])

		if key1 == key2 {
			return false
		}
	}

	return true
}
