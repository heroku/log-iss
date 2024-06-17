package telemetry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"reflect"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	// TracerName is the name of the library providing instrumentation. TracerName is used for tracer and instrumentation names.
	TracerName = "telemetry-go"

	// MeterName is the name of the library providing instrumentation.
	MeterName = "telemetry-go"
)

// A unique key to store the main OpenTelemetry span for per request
type mainSpanContextKey struct{}

// A unique key to store data in the context.
// Spans are write-only, so we need a place to store any additional metadata
// that we need to access during the lifecycle of a trace
type spanMetadataKey struct{}

// A struct containing metadata about
type spanMetadata struct {
	isVerbose bool
}

const verboseKey = "verbose"

// Use the WC3 tracing context standard to inherit trace context
var traceContextFormat propagation.TextMapPropagator = propagation.TraceContext{}

/*
HTTPMiddleware injects instrumentation into an HTTP handler.

Usage

	r := mux.NewRouter()
	r.Use(telemetry.HTTPMiddleware)
*/
func HTTPMiddleware(handler http.Handler) http.Handler {
	handle := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scrubbedURL := scrubURL(r.URL).String()
		ctx := r.Context()

		// Retrieve the span created by otelhttp.Handler
		span := trace.SpanFromContext(ctx)
		span.SetAttributes(
			semconv.HTTPTargetKey.String(scrubbedURL),
			attribute.String("request_id", r.Header.Get("X-Request-ID")),
			semconv.HTTPHostKey.String(r.Host),
		)

		ctx, span = decideVerbosity(ctx, span)
		markAsMainSpan(span)

		// store the span as an object on the request context using a private key
		ctx = context.WithValue(ctx, mainSpanContextKey{}, span)
		r = r.WithContext(ctx)

		handler.ServeHTTP(w, r)
	})

	options := []otelhttp.Option{
		otelhttp.WithPropagators(traceContextFormat),
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.URL.Path
		}),
		otelhttp.WithTracerProvider(TracerProvider()),
	}

	if config.Filter != nil {
		options = append(options, otelhttp.WithFilter(config.Filter))
	}

	return otelhttp.NewHandler(handle, "http", options...)
}

// TracerProvider is a factory for Tracers and is initialized once when you call telemetry.Configure().
// TracerProvider's lifecycle matches the application’s lifecycle.
// If a TracerProvider isn't set, telemetry.Configure has not been called (common, say, during testing) and in those cases, a default TracerProvider is set.
func TracerProvider() *sdktrace.TracerProvider {
	if config.TracerProvider == nil {
		config.TracerProvider = sdktrace.NewTracerProvider()
	}
	return config.TracerProvider
}

// Tracer creates spans containing more information about what is happening for a given operation, such as a request in a service.
// Tracers are created by TracerProvider.
func Tracer() trace.Tracer {
	return TracerProvider().Tracer(TracerName)
}

/*
StartSpan creates a normal span. A span represent a unit of work or operation by wraping code execution.
They can and should be annotated with attributes adding details about what happened during that execution.
Avoid creating too many spans. Aim for one per request/job/unit-of-work and one per asynchronous operation.
Most of these should be created by instrumentation, but sometimes you may need to create you own spans.
If trying to decide if something should be an attribute or a separate span, lean towards an attribute on an existing span.

Spans have the following properties:

	Name
	Parent span ID (empty for root spans)
	Start and End Timestamps
	Span Context
	Attributes
	Span Events
	Span Links
	Span Status

Usage:

	span := telemetry.StartSpan(ctx, "span-name")
	defer span.End()
	span.setAttributes(
		attribute.String("user_id", userID),
	)
*/
func StartSpan(ctx context.Context, name string, o ...trace.SpanStartOption) (context.Context, trace.Span) {
	return Tracer().Start(ctx, name, o...)
}

// StartMainSpan starts a new trace and marks it as a main one.
func StartMainSpan(ctx context.Context, name string, o ...trace.SpanStartOption) (context.Context, trace.Span) {
	ctx, span := Tracer().Start(ctx, name, o...)
	ctx, span = decideVerbosity(ctx, span)
	markAsMainSpan(span)
	ctx = context.WithValue(ctx, mainSpanContextKey{}, span)

	return ctx, span
}

func markAsMainSpan(span trace.Span) {
	span.SetAttributes(
		attribute.Bool(logExporterMainKey, true),
	)
	cs := ReadContainerStats()
	span.SetAttributes(
		attribute.Int64("metrics.goroutines_count", cs.NumGoroutines),
		attribute.Float64("metrics.mem_used_pct", math.Round(cs.MemUsed*100)/100),
		attribute.Float64("metrics.load_average_1m", math.Round(cs.LoadAvg1m*100)/100),
		attribute.String("metrics.last_refresh_at", cs.LastRefreshed.Format(time.RFC3339)),
		attribute.Float64("uptime_sec", time.Since(config.StartTime).Seconds()),
		attribute.Float64("uptime_sec_log10", math.Log10(time.Since(config.StartTime).Seconds())),
		attribute.Int64("metrics.gc_pause_count", cs.NumGCDelta),
		attribute.Int64("metrics.gc_pause_time_ns", cs.NumPauseDelta),
		attribute.Int64("metrics.heap_in_use_bytes", int64(cs.HeapInuse)),
		attribute.Int64("metrics.heap_objects_count", int64(cs.HeapObjects)),
		attribute.Int64("metrics.stack_in_use_bytes", int64(cs.StackInuse)),
		attribute.Int64("metrics.stack_sys_memory_bytes", int64(cs.StackSys)),
		attribute.Int64("metrics.sys_memory_bytes", int64(cs.Sys)),
	)
	if cs.DiskUsed != diskUsageError {
		span.SetAttributes(attribute.Float64("metrics.disk_used_pct", math.Round(cs.DiskUsed*100)/100))
	}

	if cs.Elapsed > 0 {
		span.SetAttributes(attribute.Float64("metrics.elapsed_ms", float64(cs.Elapsed)/float64(time.Millisecond)))
	}
}

// StartAsyncSpan starts a new trace with the parent context detached.
func StartAsyncSpan(ctx context.Context, name string, o ...trace.SpanStartOption) (context.Context, trace.Span) {
	ctx = DetachContext(ctx)
	return Tracer().Start(ctx, name, o...)
}

func decideVerbosity(ctx context.Context, span trace.Span) (context.Context, trace.Span) {
	meta := spanMetadata{
		isVerbose: rand.Float64() < config.VerboseSamplingRate,
	}
	ctx = context.WithValue(ctx, spanMetadataKey{}, meta)

	// Set verbose = true on the main span to ease finding verbose traces
	span.SetAttributes(
		attribute.Bool(verboseKey, meta.isVerbose),
	)

	return ctx, span
}

// StartVerboseSpan will condtionally create a span based on whether this trace
// is evaluated to be "verbose" or not. This allows you to wrap some operations
// in spans that might be otherwise too costly too instrument for every operation.
// You can think of verbose spans as equivalent to log_level=verbose.
func StartVerboseSpan(ctx context.Context, name string, o ...trace.SpanStartOption) (context.Context, trace.Span) {
	meta, ok := ctx.Value(spanMetadataKey{}).(spanMetadata)
	if !ok {
		return trace.NewNoopTracerProvider().Tracer("noop").Start(ctx, "noop")
	}

	if !meta.isVerbose {
		return trace.NewNoopTracerProvider().Tracer("noop").Start(ctx, "noop")
	}

	return Tracer().Start(ctx, name, o...)
}

// MainSpanFromContext returns the started main span for the current context
func MainSpanFromContext(ctx context.Context) trace.Span {
	span, ok := ctx.Value(mainSpanContextKey{}).(trace.Span)
	if !ok {
		_, span = trace.NewNoopTracerProvider().Tracer("noop").Start(ctx, "noop")
	}

	return span
}

// SetError sets the "error" attribute on the main span extracted from the context passed in.
// The error's type is set to the error value's Go package and type.
// The span's status will also indicate an error along with the error string passed in.
func SetError(ctx context.Context, err error, errorString string, kv ...attribute.KeyValue) {
	span := MainSpanFromContext(ctx)

	// set error attributes on the main span
	span.SetAttributes(
		attribute.String("error", errorString),
	)

	// use reflection to find a string representation of the error if possible
	if err != nil {
		errType := reflect.TypeOf(err)
		errTypeString := fmt.Sprintf("%s.%s", errType.PkgPath(), errType.Name())
		if errTypeString == "." {
			errTypeString = errType.String()
		}
		span.SetAttributes(
			attribute.String("error.type", errTypeString),
		)
	}

	// Add any additional attributes passed in to the main span
	for _, att := range kv {
		span.SetAttributes(att)
	}

	// record an error
	// this will add an error event to the span and set `status.code`
	span.RecordError(err)
	span.SetStatus(codes.Error, errorString)
}

// HTTP Transport RoundTripper meant to be passed to OpenTelemetry's RoundTripper
// instead of the default implementation.
//
// It is important that this be passed to OpenTelemetry's RoundTripper and not
// vice-versa so that this RoundTripper can have access to the span created
// by OpenTelemetry's RT.
type transport struct {
	rt http.RoundTripper
}

// RoundTrip is used for stripping outgoing requests of sensitive information.
func (t *transport) RoundTrip(r *http.Request) (*http.Response, error) {
	// Pull out the current span and overwrite http.url with a scrubbed version
	span := trace.SpanFromContext(r.Context())
	scrubbedURL := scrubURL(r.URL).String()
	span.SetAttributes(
		semconv.HTTPURLKey.String(scrubbedURL),
	)

	// If we've added an annotator to the request, pull out any information inside
	// and add it to the span
	if annotator, ok := annotatorFromContext(r.Context()); ok {
		if name := annotator.getName(); name != "" {
			span.SetName(name)
		}

		attrs := annotator.getAttributes()
		span.SetAttributes(attrs...)
	}

	// delegate to the original RoundTripper
	return t.rt.RoundTrip(r)
}

func HTTPTransport(base http.RoundTripper) http.RoundTripper {
	return otelhttp.NewTransport(
		&transport{
			rt: base,
		},
		otelhttp.WithPropagators(traceContextFormat),
	)
}

func HTTPTransportWithoutTraceContextPropagation(base http.RoundTripper) http.RoundTripper {
	return otelhttp.NewTransport(
		&transport{
			rt: base,
		},
	)
}

func AnnotateRequest(r *http.Request, name string, attrs ...attribute.KeyValue) *http.Request {
	annotator, ok := annotatorFromContext(r.Context())

	// If there is no existing annotator in the request context, add one
	if !ok {
		ctx := injectAnnotator(r.Context(), annotator)
		r = r.WithContext(ctx)
	}

	annotator.setName(name)
	annotator.addAttributes(attrs...)

	return r
}

func GetSerializedTraceContext(ctx context.Context) ([]byte, error) {
	traceContextTextMap := newTextMap()
	traceContextFormat.Inject(ctx, &traceContextTextMap)
	return traceContextTextMap.Serialize()
}

// SetSerializedTraceContext generates a new context that inherits from this serialized trace context.
func SetSerializedTraceContext(ctx context.Context, serialized []byte) (context.Context, error) {
	traceContextTextMap, err := newTextMapFromSerialized(serialized)
	if err != nil {
		return ctx, err
	}
	ctx = traceContextFormat.Extract(ctx, traceContextTextMap)

	// assert that the trace context is valid
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return ctx, errors.New("Invalid span context")
	}

	return ctx, nil
}

type granularity int64

const (
	nanoseconds = iota
	milliseconds
	seconds
)

// Timer holds values to set a timing to the current span.
// You should use TimerSeconds, TimerMilliseconds or TimerNanoseconds to instantiate a Timer.
type Timer struct {
	ctx         context.Context
	name        string
	granularity granularity
	startTime   time.Time
}

/*
End ends a Timer.

Usage:

	timer := TimerNanoseconds(ctx, "work_duration_ns")
	...
	timer.End()
*/
func (t *Timer) End() {
	span := trace.SpanFromContext(t.ctx)

	switch t.granularity {

	case nanoseconds:
		span.SetAttributes(
			attribute.Float64(t.name, time.Since(t.startTime).Seconds()),
		)
	case milliseconds:
		span.SetAttributes(
			attribute.Int64(t.name, time.Since(t.startTime).Milliseconds()),
		)
	case seconds:
		span.SetAttributes(
			attribute.Int64(t.name, time.Since(t.startTime).Nanoseconds()),
		)
	default:
		span.AddEvent("unknown-granularity", trace.WithAttributes(
			attribute.Int("granularity", int(t.granularity)),
			attribute.String("timing_name", t.name),
		))
	}
}

// this function will add a timing to the current span
func setTiming(ctx context.Context, name string, g granularity) Timer {
	timer := Timer{ctx, name, g, time.Now()}

	return timer
}

/*
TimerSeconds is a helper used to set timing to the current span.

Usage:

	timer := TimerSeconds(ctx, "work_duration_seconds")
	...
	timer.End()
*/
func TimerSeconds(ctx context.Context, name string) Timer {
	return setTiming(ctx, name, seconds)
}

/*
TimerMilliseconds is a helper used to set timing to the current span.

Usage:

	timer := TimerMilliseconds(ctx, "work_duration_ms")
	...
	timer.End()
*/
func TimerMilliseconds(ctx context.Context, name string) Timer {
	return setTiming(ctx, name, milliseconds)
}

/*
TimerNanoseconds is a helper used to set timing to the current span.

Usage:

	timer := TimerNanoseconds(ctx, "work_duration_ns")
	...
	timer.End()
*/
func TimerNanoseconds(ctx context.Context, name string) Timer {
	return setTiming(ctx, name, nanoseconds)
}
