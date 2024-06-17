package telemetry

import (
	"bytes"
	"context"
	"io"
	"sort"
	"time"

	"github.com/go-logfmt/logfmt"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

/*
LogExporter is an opinionated OpenTelemetry exporter

Rather than marshalling the collected spans to a third-party service,
this encourages the use of the "canonical-log-line" pattern (https://brandur.org/canonical-log-lines).

Here, we allow certain spans to be identified as "canonical" which we're
calling "main" by setting a special `logExporterMainKey` as an attribute.
These spans are meant to span the main "units-of-work" of the app, usually
a req / res cycle and collect all of the relevant metadata for that request.

Examples of metadata:
- user id
- enterprise id
- team id
- error message
- elapsed time

There are a couple of ideas that are important to grasp here:
- Structured log lines are easier to process than arbitrarily formatted text
- Logs are only incidentally meant to be read by humans
- OpenTelemetry spans can be thought of as Events that contain keys and values
- Splunk's query language allows us to treat each log-line like a table row in a database
- The more context we can add to one line, the more questions we can ask of the data later. Ex: Is this spike in p99 response time due to one user, or many? Which one? Is it one API endpoint or many of them?
*/
type LogExporter struct {
	writer io.Writer
}

// Compile time assertion that the exporter implements sdktrace.Exporter
var _ sdktrace.SpanExporter = (*LogExporter)(nil)

const logExporterMainKey = "main"

// ExportSpan converts each span into a single structured log-line in logfmt format
func (le *LogExporter) ExportSpans(ctx context.Context, s []sdktrace.ReadOnlySpan) error {
	for _, v := range s {
		err := le.exportSingleSpan(ctx, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (le *LogExporter) exportSingleSpan(ctx context.Context, spanData sdktrace.ReadOnlySpan) error {
	if isMainSpan(spanData) {
		var buf bytes.Buffer
		encoder := logfmt.NewEncoder(&buf)

		spanContext := spanData.SpanContext()

		fields := map[string]interface{}{
			"trace_id":    spanContext.TraceID().String(),
			"span_id":     spanContext.SpanID().String(),
			"name":        spanData.Name(),
			"start_time":  spanData.StartTime().Format(time.RFC3339Nano),
			"end_time":    spanData.EndTime().Format(time.RFC3339Nano),
			"status_code": spanData.Status().Code,
		}

		it := spanData.Resource().Iter()
		for it.Next() {
			attr := it.Attribute()
			fields[string(attr.Key)] = attr.Value.AsInterface()
		}

		if spanData.Status().Description != "" {
			fields["status_description"] = spanData.Status().Description
		}

		// If we have exceeded our MaxAttributesPerSpan configuration, log it out
		// so we can track it
		if spanData.DroppedAttributes() > 0 {
			fields["dropped_attribute_count"] = spanData.DroppedAttributes()
		}

		// A span without a parent id will print an empty value as "0000000000000000"
		// so we exclude it
		if spanData.Parent().SpanID() != (trace.SpanID{}) {
			fields["parent_span_id"] = spanData.Parent().SpanID().String()
		}

		if start, end := spanData.StartTime(), spanData.EndTime(); !start.IsZero() && !end.IsZero() {
			fields["duration_ms"] = float64(end.Sub(start)) / float64(time.Millisecond)
		}

		if len(spanData.Attributes()) != 0 {
			for _, kv := range spanData.Attributes() {
				fields[string(kv.Key)] = kv.Value.AsInterface()
			}
		}

		// collect all keys and sort them
		keys := make([]string, len(fields))
		i := 0
		for key := range fields {
			keys[i] = key
			i++
		}
		sort.Strings(keys)

		// convert to logfmt
		for _, key := range keys {
			err := encoder.EncodeKeyval(key, fields[key])
			if err != nil {
				return err
			}
		}

		err := encoder.EndRecord()
		if err != nil {
			return err
		}

		if le.writer != nil {
			_, err = le.writer.Write(buf.Bytes())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (le *LogExporter) Shutdown(ctx context.Context) error {
	return nil
}

func isMainSpan(s sdktrace.ReadOnlySpan) bool {
	for _, kv := range s.Attributes() {
		if kv.Key == logExporterMainKey {
			return true
		}
	}
	return false
}
