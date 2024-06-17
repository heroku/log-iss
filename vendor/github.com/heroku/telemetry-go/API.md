# Package telemery-go API Docs

Package telemetry is the Go implementation of the Heroku/Evergreen Platform Observability configuration.
The library provides a set of integrations to allow easy generation of standardized telemetry from Heroku and Evergreen components.

# Configuration

```go
err := telemetry.Configure(
	telemetry.WithConfig(telemetry.Config{
		Component:   "<YOUR COMPONENT NAME>",
		Environment: os.Getenv("ENVIRONMENT"),
		Version:     "<THE VERSION CURRENTLY RUNNING>",
		Instance:    "<DYNO ID OR AWS INSTANCE ID>",
	}),
	telemetry.WithHoneycomb("<HONEYCOMB API KEY>", "<HONEYCOMB DATASET>"),
})

if err != nil {
	// ...
}

defer telemetry.Close()
```

## Constants

```golang
const (
    // TracerName is the name of the library providing instrumentation. TracerName is used for tracer and instrumentation names.
    TracerName = "telemetry-go"

    // MeterName is the name of the library providing instrumentation.
    MeterName = "telemetry-go"
)
```

## Functions

### func [AnnotateRequest](/telemetry.go#L311)

`func AnnotateRequest(r *http.Request, name string, attrs ...attribute.KeyValue) *http.Request`

### func [Close](/configure.go#L435)

`func Close()`

Close closes all exporters for shutdown.

### func [Configure](/configure.go#L270)

`func Configure(opts ...ConfigOption) error`

Configure configures all the observability components required.
Defaults include sampling rate, verbose sampling rate, writing logs to STDOUT,
Heroku dyno metadata as global attributes (see [https://devcenter.heroku.com/articles/dyno-metadata](https://devcenter.heroku.com/articles/dyno-metadata)),
and more.

### func [DetachContext](/xcontext.go#L19)

`func DetachContext(ctx context.Context) context.Context`

Detach returns a context that keeps all the values of its parent context
but detaches from the cancellation and error handling.

### func [ExtractAttributesFromSpan](/test_helpers.go#L8)

`func ExtractAttributesFromSpan(s trace.ReadOnlySpan) map[string]attribute.Value`

### func [Float64Count](/metric.go#L38)

`func Float64Count(ctx context.Context, name string, value float64)`

Float64Count recording increasing values.

### func [Float64Histogram](/metric.go#L74)

`func Float64Histogram(ctx context.Context, name string, value float64)`

Float64Histogram recording a distribution of values.

### func [Float64UpDownCount](/metric.go#L47)

`func Float64UpDownCount(ctx context.Context, name string, value float64)`

Float64UpDownCount recording changes of a value.

### func [GetSerializedTraceContext](/telemetry.go#L326)

`func GetSerializedTraceContext(ctx context.Context) ([]byte, error)`

### func [HTTPMiddleware](/telemetry.go#L56)

`func HTTPMiddleware(handler http.Handler) http.Handler`

HTTPMiddleware injects instrumentation into an HTTP handler.

Usage

```go
r := mux.NewRouter()
r.Use(telemetry.HTTPMiddleware)
```

### func [HTTPTransport](/telemetry.go#L294)

`func HTTPTransport(base http.RoundTripper) http.RoundTripper`

### func [HTTPTransportWithoutTraceContextPropagation](/telemetry.go#L303)

`func HTTPTransportWithoutTraceContextPropagation(base http.RoundTripper) http.RoundTripper`

### func [Int64Count](/metric.go#L29)

`func Int64Count(ctx context.Context, name string, value int64)`

Int64Count recording increasing values.

### func [Int64Histogram](/metric.go#L65)

`func Int64Histogram(ctx context.Context, name string, value int64)`

Int64Histogram recording a distribution of values.

### func [Int64UpDownCount](/metric.go#L56)

`func Int64UpDownCount(ctx context.Context, name string, value int64)`

Int64UpDownCount recording changes of a value.

### func [MainSpanFromContext](/telemetry.go#L217)

`func MainSpanFromContext(ctx context.Context) trace.Span`

MainSpanFromContext returns the started main span for the current context

### func [Meter](/metric.go#L21)

`func Meter() metricApi.Meter`

Meter provides access to instrument instances for recording metrics.

### func [MeterProvider](/metric.go#L13)

`func MeterProvider() *metric.MeterProvider`

MeterProvider returns the registered global meter provider.
If none is registered then a No-op MeterProvider is returned.

### func [SetError](/telemetry.go#L229)

`func SetError(ctx context.Context, err error, errorString string, kv ...attribute.KeyValue)`

SetError sets the "error" attribute on the main span extracted from the context passed in.
The error's type is set to the error value's Go package and type.
The span's status will also indicate an error along with the error string passed in.

### func [SetSerializedTraceContext](/telemetry.go#L333)

`func SetSerializedTraceContext(ctx context.Context, serialized []byte) (context.Context, error)`

SetSerializedTraceContext generates a new context that inherits from this serialized trace context.

### func [StartAsyncSpan](/telemetry.go#L180)

`func StartAsyncSpan(ctx context.Context, name string, o ...trace.SpanStartOption) (context.Context, trace.Span)`

StartAsyncSpan starts a new trace with the parent context detached.

### func [StartMainSpan](/telemetry.go#L141)

`func StartMainSpan(ctx context.Context, name string, o ...trace.SpanStartOption) (context.Context, trace.Span)`

StartMainSpan starts a new trace and marks it as a main one.

### func [StartSpan](/telemetry.go#L136)

`func StartSpan(ctx context.Context, name string, o ...trace.SpanStartOption) (context.Context, trace.Span)`

StartSpan creates a normal span. A span represent a unit of work or operation by wraping code execution.
They can and should be annotated with attributes adding details about what happened during that execution.
Avoid creating too many spans. Aim for one per request/job/unit-of-work and one per asynchronous operation.
Most of these should be created by instrumentation, but sometimes you may need to create you own spans.
If trying to decide if something should be an attribute or a separate span, lean towards an attribute on an existing span.

Spans have the following properties:

```go
Name
Parent span ID (empty for root spans)
Start and End Timestamps
Span Context
Attributes
Span Events
Span Links
Span Status
```

Usage:

```go
span := telemetry.StartSpan(ctx, "span-name")
defer span.End()
span.setAttributes(
	attribute.String("user_id", userID),
)
```

### func [StartVerboseSpan](/telemetry.go#L203)

`func StartVerboseSpan(ctx context.Context, name string, o ...trace.SpanStartOption) (context.Context, trace.Span)`

StartVerboseSpan will condtionally create a span based on whether this trace
is evaluated to be "verbose" or not. This allows you to wrap some operations
in spans that might be otherwise too costly too instrument for every operation.
You can think of verbose spans as equivalent to log_level=verbose.

### func [Tracer](/telemetry.go#L106)

`func Tracer() trace.Tracer`

Tracer creates spans containing more information about what is happening for a given operation, such as a request in a service.
Tracers are created by TracerProvider.

### func [TracerProvider](/telemetry.go#L97)

`func TracerProvider() *sdktrace.TracerProvider`

TracerProvider is a factory for Tracers and is initialized once when you call telemetry.Configure().
TracerProvider's lifecycle matches the application’s lifecycle.
If a TracerProvider isn't set, telemetry.Configure has not been called (common, say, during testing) and in those cases, a default TracerProvider is set.

## Types

### type [Config](/configure.go#L87)

`type Config struct { ... }`

Config is a user-facing set of confuration data that every service should
try to fill out

### type [ConfigOption](/configure.go#L44)

`type ConfigOption func(*TelemetryConfig) error`

#### func [WithConfig](/configure.go#L118)

`func WithConfig(c Config) ConfigOption`

WithConfig lets you specify ConfigOptions with initializing the telemetry-go library.

#### func [WithDiskRoot](/configure.go#L188)

`func WithDiskRoot(root string) ConfigOption`

WithDiskRoot specifies the directory root config option.

#### func [WithFilter](/configure.go#L259)

`func WithFilter(filter otelhttp.Filter) ConfigOption`

WithFilter allows a function to be configured that will
be called on every request. If the filter function returns
false that request will not be traced. Note that this may
be the opposite of what you expect.

This is helpful to avoid tracing requests like health
checks that are of relatively little utility. To exclude
only the /health route you could pass the following
configuration:

```go
telemetry.WithFilter(func(r *http.Request) bool {
 	if r != nil {
 		return r.URL.Path != "/health"
 	}
 	return true
}),
```

#### func [WithGlobalAttributes](/configure.go#L214)

`func WithGlobalAttributes(attrs ...attribute.KeyValue) ConfigOption`

WithGlobalAttributes lets you specify the key/value attributes that should be on every trace.

#### func [WithHoneycomb](/configure.go#L153)

`func WithHoneycomb(apiKey string, dataset string) ConfigOption`

WithHoneycomb lets you specify the API Key and Dataset to use when relaying data to Honeycomb.

#### func [WithHoneycombApiKey](/configure.go#L163)

`func WithHoneycombApiKey(apiKey string) ConfigOption`

WithHoneycombApiKey lets you specify the API Key to use when relaying data to Honeycomb.

#### func [WithLogWriter](/configure.go#L206)

`func WithLogWriter(writer io.Writer) ConfigOption`

WithLogWriter lets yous specify your own log writer.

#### func [WithMetricsDatasetHoneycomb](/configure.go#L172)

`func WithMetricsDatasetHoneycomb(dataset string) ConfigOption`

WithMetricsDatasetHoneycomb lets you specify the Metrics Dataset to use when relaying data to Honeycomb.

#### func [WithMetricsEnabled](/configure.go#L145)

`func WithMetricsEnabled() ConfigOption`

WithMetricsEnabled will enable send metrics to Honeycomb.

#### func [WithSamplingRate](/configure.go#L228)

`func WithSamplingRate(rate float64) ConfigOption`

WithSamplingRate lets you override the default sampling rate.

#### func [WithTestExporter](/configure.go#L197)

`func WithTestExporter(exporter *TestExporter) ConfigOption`

WithTestExportedr lets you specify your own exporter for testing purposes.
Sampling is turned off by default when testing.

#### func [WithTracesDatasetHoneycomb](/configure.go#L180)

`func WithTracesDatasetHoneycomb(dataset string) ConfigOption`

WithTracesDatasetHoneycomb lets you specify the Traces Dataset to use when relaying data to Honeycomb.

#### func [WithVerboseSamplingRate](/configure.go#L236)

`func WithVerboseSamplingRate(rate float64) ConfigOption`

WithVerboseSamplingRate lets you override the default verbose sampling rate.

### type [ContainerStats](/container_stats.go#L41)

`type ContainerStats struct { ... }`

ContainerStats holds the container information at a given time

#### func [ReadContainerStats](/container_stats.go#L58)

`func ReadContainerStats() ContainerStats`

ReadContainerStats reads the latest container stats values

### type [HerokuAppMetadata](/configure.go#L46)

`type HerokuAppMetadata struct { ... }`

### type [LogExporter](/logexporter.go#L40)

`type LogExporter struct { ... }`

LogExporter is an opinionated OpenTelemetry exporter

Rather than marshalling the collected spans to a third-party service,
this encourages the use of the "canonical-log-line" pattern ([https://brandur.org/canonical-log-lines](https://brandur.org/canonical-log-lines)).

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

#### func (*LogExporter) [ExportSpans](/logexporter.go#L50)

`func (le *LogExporter) ExportSpans(ctx context.Context, s []sdktrace.ReadOnlySpan) error`

ExportSpan converts each span into a single structured log-line in logfmt format

#### func (*LogExporter) [Shutdown](/logexporter.go#L141)

`func (le *LogExporter) Shutdown(ctx context.Context) error`

### type [TelemetryConfig](/configure.go#L60)

`type TelemetryConfig struct { ... }`

TelemetryConfig holds all the global values configured for this component

### type [TestExporter](/testexporter.go#L13)

`type TestExporter struct { ... }`

TestExporter exporter for collecting spans in memory for testing

#### func (*TestExporter) [Aggregation](/testexporter.go#L23)

`func (t *TestExporter) Aggregation(kind metric.InstrumentKind) metric.Aggregation`

#### func (*TestExporter) [ClearRecordedSpans](/testexporter.go#L59)

`func (t *TestExporter) ClearRecordedSpans()`

#### func (*TestExporter) [Export](/testexporter.go#L63)

`func (t *TestExporter) Export(ctx context.Context, resourceMetrics *metricdata.ResourceMetrics) error`

#### func (*TestExporter) [ExportSpans](/testexporter.go#L31)

`func (t *TestExporter) ExportSpans(ctx context.Context, s []trace.ReadOnlySpan) error`

#### func (*TestExporter) [ForceFlush](/testexporter.go#L27)

`func (t *TestExporter) ForceFlush(ctx context.Context) error`

#### func (*TestExporter) [GetRecordedMainSpans](/testexporter.go#L39)

`func (t *TestExporter) GetRecordedMainSpans() []trace.ReadOnlySpan`

#### func (*TestExporter) [GetRecordedMetrics](/testexporter.go#L71)

`func (t *TestExporter) GetRecordedMetrics() *metricdata.ResourceMetrics`

#### func (*TestExporter) [GetRecordedSpans](/testexporter.go#L55)

`func (t *TestExporter) GetRecordedSpans() []trace.ReadOnlySpan`

#### func (*TestExporter) [Shutdown](/testexporter.go#L77)

`func (t *TestExporter) Shutdown(ctx context.Context) error`

#### func (*TestExporter) [Temporality](/testexporter.go#L19)

`func (t *TestExporter) Temporality(kind metric.InstrumentKind) metricdata.Temporality`

### type [Timer](/telemetry.go#L359)

`type Timer struct { ... }`

Timer holds values to set a timing to the current span.
You should use TimerSeconds, TimerMilliseconds or TimerNanoseconds to instantiate a Timer.

#### func [TimerMilliseconds](/telemetry.go#L429)

`func TimerMilliseconds(ctx context.Context, name string) Timer`

TimerMilliseconds is a helper used to set timing to the current span.

Usage:

```go
timer := TimerMilliseconds(ctx, "work_duration_ms")
...
timer.End()
```

#### func [TimerNanoseconds](/telemetry.go#L442)

`func TimerNanoseconds(ctx context.Context, name string) Timer`

TimerNanoseconds is a helper used to set timing to the current span.

Usage:

```go
timer := TimerNanoseconds(ctx, "work_duration_ns")
...
timer.End()
```

#### func [TimerSeconds](/telemetry.go#L416)

`func TimerSeconds(ctx context.Context, name string) Timer`

TimerSeconds is a helper used to set timing to the current span.

Usage:

```go
timer := TimerSeconds(ctx, "work_duration_seconds")
...
timer.End()
```

#### func (*Timer) [End](/telemetry.go#L375)

`func (t *Timer) End()`

End ends a Timer.

Usage:

```go
timer := TimerNanoseconds(ctx, "work_duration_ns")
...
timer.End()
```
