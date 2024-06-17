package telemetry

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
)

func ExtractAttributesFromSpan(s trace.ReadOnlySpan) map[string]attribute.Value {
	data := map[string]attribute.Value{}
	for _, kv := range s.Attributes() {
		data[string(kv.Key)] = kv.Value
	}

	it := s.Resource().Iter()
	for it.Next() {
		attr := it.Attribute()
		data[string(attr.Key)] = attr.Value
	}
	return data
}
