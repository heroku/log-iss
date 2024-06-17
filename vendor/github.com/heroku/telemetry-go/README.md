# Telemetry

_Please reach out to the Platform Observability team in #dx-pe-observability before utilizing this library. This documentation is not yet meant to be self-service. We are engaging with each interested team directly to walk them through the on-boarding process and get necessary feedback for future self-service documentation._

This is the Go implementation of the Heroku/Evergreen Platform Observability configuration. The library provides a set of integrations to allow easy generation of standardized telemetry from Heroku and Evergreen components.

## Installation

```
GOPRIVATE=github.com/heroku go get github.com/heroku/telemetry-go
```

Since this repository is private, you should vendor the dependency to be able to use it in non-local environments.

## API Documentation

See [API.md](API.md).

## Configuration

Configure telemetry within each of your service's binaries

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

## Instrumenting an HTTP Server

There are two steps to instrumenting a go HTTP service. Incoming requests need to be wrapped so that
trace http headers can be picked up for distributed traces and generating the `main` span for the
request.

Likewise outgoing requests to other services (Ex: querying the Heroku API) need to be instrumented
so that the correct trace context can be sent along as header information.

### Incoming requests

To instrument incoming requests, this library provides an HTTP Middleware function. In addition to
processing the incoming trace context, this will automatically pull out information from the request
and response. Ex: IP address, http status code, response time.

#### `gorilla/mux` router

For services using the `gorilla/mux` router, you can instrument by adding an additional HTTP Middleware

```go
	r := mux.NewRouter()

	// We recommend naming each route
	r.HandleFunc("/", handler).Name("HomeHandler")

	// Where routes contain parameters, these will automatically parsed
	// out and added to the emitted telemetry
	r.HandleFunc("/articles/{category}/{id:[0-9]+}", handler).Name("ArticleHandler")

  // For best results, add the telemetry middleware as the first
	// middleware in your middleware stack
	r.Use(telemetry.HTTPMiddleware)
```

#### http.handler

For those that do not use `mux`, you can use `telemetry.HTTPMiddleware` directly by wrapping an `http.Handler`

```go
handler = telemetry.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// ...
}))
```

### Outgoing requests

To add trace context to your outgoing requests, there are two things you need to do.

First add the `telemetry.HTTPTransport` to your `http.Client`.

```go
client := http.Client{
	Transport: telemetry.HTTPTransport(http.DefaultTransport),
}
```

If you already have a `Transport`, you can pass that in instead of `http.DefaultTransport`. It will be wrapped
with the new functionality and continue functioning as before.

```go
client := http.Client{
	Transport: telemetry.HTTPTransport(existingTransport),
}
```

Secondly, when making a request, you need to use `http.NewRequestWithContext` instead of `http.NewRequest`, and
pass in a `context.Context` with a valid trace context established. If you are using `telemetry.HTTPMiddleware`
this is likely `req.Context()`.

```go
// The transport wrapper will take care of extracting the trace context from the `context.Context` object
// and creating the correct headers
req, err := http.NewRequestWithContext(ctx, "GET", url)
if err != nil {
	// ...
}
res, err := client.Do(req)
```

### Outgoing requests without propagating trace context

While propagating trace context through http headers is what you will want most of the time, there
are situations, such as calling customer or 3rd party code where we will not want to include the 
trace context headers in our HTTP request, but will still want to track the request in our traces.

For this purpose we have provided `telemetry.HTTPTransportWithoutTraceContextPropagation` which will track
the request, but not append any additional trace context headers.

```go
client := http.Client{
	Transport: telemetry.HTTPTransportWithoutTraceContextPropagation(http.DefaultTransport),
}
```

## Enriching the default telemetry

The telemetry plugins will pull all of the data about the request that it can, however it does not
know anything about what your application is doing or how it may relate to our business. While 
processing a request we know a lot more about it than HTTP Status or how long it took! We
know what action the request is performing, who made the request, whether this is a free user or a VIP,
what errors we hit or how many times we might have had to retry an action, etc.

We should record as much of this context as we can as additional attributes on the `main` span.
At any point during the request you can add these by fetching the main span from the context object.

```go
telemetry.MainSpanFromContext(ctx).AddAtributes(
	attribute.String("request_id", requestID),
	attribute.String("user_id", userID),
	attribute.String("org_id", orgID),
	attribute.Int("auth_duration_ms", authDuration.Milliseconds()),
	attribute.Int("retry_count", numRetries),
)
```

## Sampling

We have some initial support for sampling that can be initialized with the `telemetry.WithSamplingRate`
configuration option. Currently passing in a float between `0` and `1` will determine the percentage of
traces that will be delivered.

In the example below we are requesting to receive traces for 10% of actions.

```go
	err = telemetry.Configure(
		// ...
		telemetry.WithSamplingRate(0.1),
	)
```

Currently, there is no support for passing along sampling rate information to Honeycomb, so enabling
this will distort the numbers you see in the UI in absolute terms, but will still be useful for relative
terms.

Ex: asking for the total # of requests within a time period will be off by a factor of the sampling rate
but asking for a percentage of failing requests or a percentage increase in traffic should still be accurate.

## Usage

1. At the beginning of every request, start a new main span

```
ctx, span := telemetry.StartMainSpan(context.Background(), "request")
```

2. At any point within that request, you can annotate the main span, provided you have access to the context

```
telemetry.MainSpanFromContext(ctx).AddAttributes(
	trace.Int64Attribute("http.status", resp.StatusCode),
)
```

3. At the end of the request, end the span

```
span.End()
```

## Manual Trace Context Propagation

OpenTelemetry provides plugins that can help propagate trace context for common requests like `http`
and `grpc`. If you find you need a new one, please reach out in `#pe-observability`, and we'll try to
help.

However, there will always be cases where existing libraries or plugins are not available or that may
be specific to the systems that you are working with. Here, we provide two functions to serialize / deserialize the trace context into a string that you can include manually in your payload.

```go
// get the trace context serialized into a byte array
serialized, err := telemetry.GetSerializedTraceContext(ctx)
```

```go
// generate a new context that inherits from this serialized trace context
newCtx, err = telemetry.SetSerializedTraceContext(newCtx, serialized)
```

## Keep child context alive even though parent context end
If you find yourself in a situation where a child context is closed because the parent context closed, and you need the child context to outlive the parent, you can use the `DetachContext` method.

```go
// detach context from parent context
detachedCtx := telemetry.DetachContext(ctx)
ctx, span := telemetry.StartSpan(detachedCtx, "async-operation")
```

## Metric API usage
i.e: Counter, adding 1 (int64) as value
```
telemetry.Int64Count(req.Context(), "request.logging.middleware.count", 1)
```

## Go version compatibility

Given our heavy reliance on `opentelemetry-go`, we can only commit to supporting versions of go supported by this underlying library. `opentelemetry-go` tries to track the official supported versions of go, that is, [each release is supported until there are two newer major releases](https://go.dev/doc/devel/release#policy).

You can see the current supported versions [here](https://github.com/open-telemetry/opentelemetry-go#compatibility)

## Development Setup

1. Use a recent version of Go (ideally 1.19+)
2. Install and verify linter (`make install-linter && make lint`)
3. Run tests (`make test`)
4. Run example server setup (`make setup-example-server`)
5. Run example server (`make run-example-server`)
6. Run example client once the example server is running (`make run-example-client`)

## Example Server

This repository has an example server that is instrumented with this library. You can use it to copy example code as well for testing when developing on this codebase locally. Traces are emitted to `STDOUT` as well as to Honeycomb.

### Configuration

You'll need `HONEYCOMB_API_KEY` and `HONEYCOMB_DATASET` in your environment before running the server. You can obtain these through the Honeycomb dashboard. For convenience these can be set in an `example/server/.env` file.

```
$ cp examples/server/.env.example examples/server/.env
```

> Take care to use a test key/dataset for your development work and tests, not production configs.

If you have access to the `telemetry-go-example` app on Heroku, you can do this in one command by running:

```
$ make setup-example-server
```

### Running the server
```
$ make run-example-server
{"level":"info","msg":"Server listening on port 8000","time":"2022-09-14T14:16:03-04:00"}
```

#### Call on a route
```
$ curl localhost:8000/todo  
[{"id":"1","text":"Buy milk"},{"id":"2","text":"Pick up laundry"},{"id":"3","text":"Call mom"}]
```
> Future updates will have a client you can execute that exercises all the routes, perhaps multiple times to generate lots of traces for debugging/testing.

Structured logging is employed. Traces will be emitted to `STDOUT`:

```
{"level":"info","msg":"Server listening on port 8000","time":"2022-09-14T14:16:03-04:00"}
{"level":"debug","method":"GET","msg":"","request-id":"212abccf-e07b-4b3f-a350-aeeed2df3944","time":"2022-09-14T14:16:12-04:00","uri":"/todo"}
deployment.environment=test duration_ms=0.49275 end_time=2022-09-14T14:16:12.24195475-04:00 host.name=localhost http.flavor=1.1 http.host=localhost:8000 http.method=GET http.scheme=http http.server_name=http http.status_code=200 http.target=/todo http.user_agent=curl/7.79.1 http.wrote_bytes=96 main=true metrics.disk_used_pct=19.01 metrics.elapsed_ms=0.202625 metrics.gc_pause_count=0 metrics.gc_pause_time_ns=0 metrics.goroutines_count=9 metrics.heap_in_use_bytes=3440640 metrics.heap_objects_count=9599 metrics.last_refresh_at=2022-09-14T14:16:12-04:00 metrics.load_average_1m=4.41 metrics.mem_used_pct=55.16 metrics.stack_in_use_bytes=1015808 metrics.stack_sys_memory_bytes=1015808 metrics.sys_memory_bytes=0 name=/todo net.host.name=localhost net.host.port=8000 net.peer.ip=127.0.0.1 net.peer.port=53156 net.transport=ip_tcp request_id= service.component=telemetry-go-example-server service.name="telemetry-go-example-server - test" service.team=observability service.version=v1 span_id=3d93aedc52fe3aca start_time=2022-09-14T14:16:12.241462-04:00 status_code=Unset telemetry.sdk.language=go telemetry.sdk.name=github.com/heroku/telemetry-go telemetry.sdk.version=(devel) trace_id=48319f7955e01c4cf19fe33bcf1bc353 uptime_sec=8.395610333 uptime_sec_log10=0.9240522776822826 verbose=false
```

### Example App on Heroku

The app can also run on and is available on the Heroku platform: https://telemetry-go-example.herokuapp.com

You can reach it like you can the location version via a client like `cURL`:

```
$ curl https://telemetry-go-example.herokuapp.com/todo
{"1":{"id":"1","text":"Buy milk"},"2":{"id":"2","text":"Pick up laundry"},"3":{"id":"3","text":"Call mom"}}
```

You can tail the app's logs as you do any other Heroku app with `h logs -t -a telemetry-go-example`.

#### Configuration

The following environment variables are already configured as env vars: `HONEYCOMB_API_KEY`, `HONEYCOMB_DATASET` (see app's Settings on its [Heroku Dashboard](https://dashboard.heroku.com/apps/telemetry-go-example/settings)).

The `HONEYCOMB_API_KEY` setting's value is available through [Honeycomb](https://ui.honeycomb.io/heroku/environments/$legacy$/api_keys) and the key is named `telemetry-go-example`.

#### Credroll

1. Login to Honeycomb Dashboard's [Settings](https://ui.honeycomb.io/heroku/environments/$legacy$/api_keys)
2. Create a new API Key, name it something akin to `telemetry-go-example` (e.g. `telemetry-go-example-1`)
3. Locate key named `telemetry-go-example`, click edit, uncheck "Enable"
4. Copy the new API key
5. Update the Heroku app's config: `h config:set -a telemetry-go-example HONEYCOMB_API_KEY=<NEW_KEY>`

#### Deployment

Run `make deploy-example-server`. This will do the following:

1. Performs a docker build using `Dockerfile.example.server`
2. The server binary will be build inside its own container (multi-stage build).
3. The image will be tagged and scanned (with snyk).
4. The image will be pushed to the Heroku registry (registry.heroku.com)
5. The image will be released via a `heroku container:release` command.

Note that during the process you may be asked to perform a login for the Heroku container registry.