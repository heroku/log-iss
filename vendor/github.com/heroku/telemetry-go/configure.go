package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	metricApi "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/heroku/telemetry-go/internal/sdk"

	"github.com/joeshaw/envdecode"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/grpc/credentials"
)

const maxAttributesPerSpan int = 100
const defaultSamplingRate float64 = 1.0
const defaultVerboseSamplingRate float64 = 0.1
const defaultDiskRootValue string = "/"
const reportingDataset string = "o11y-tracking"
const defaultHoneycombApiKey = "TELEMETRY_HONEYCOMB_API_KEY"

type ConfigOption func(*TelemetryConfig) error

type HerokuAppMetadata struct {
	Dyno             string `env:"DYNO"`
	DynoType         string
	DynoIndex        int
	AppID            string `env:"HEROKU_APP_ID"`
	DynoID           string `env:"HEROKU_DYNO_ID"`
	ReleaseVersion   string `env:"HEROKU_RELEASE_VERSION"`
	AppName          string `env:"HEROKU_APP_NAME"`
	ReleaseCreatedAt string `env:"HEROKU_RELEASE_CREATED_AT"`
	SlugCommit       string `env:"HEROKU_SLUG_COMMIT"`
	SlugDescription  string `env:"HEROKU_SLUG_DESCRIPTION"`
}

// TelemetryConfig holds all the global values configured for this component
type TelemetryConfig struct {
	// SamplingRate >= 1 will always sample. SamplingRate < 0 is treated as zero.
	SamplingRate            float64
	VerboseSamplingRate     float64
	UserConfig              Config
	Resource                *resource.Resource
	LogWriter               io.Writer
	HoneycombAPIKey         string
	HoneycombTracesDataset  string
	HoneycombMetricsDataset string
	MetricsEnabled          bool
	CollectorURL            string
	CollectorToken          string
	DiskRoot                string
	TestExporter            *TestExporter
	TelemetryAddonEnabled   bool
	Filter                  otelhttp.Filter
	TracerProvider          *sdktrace.TracerProvider
	MeterProvider           *metric.MeterProvider
	Meter                   metricApi.Meter
	StartTime               time.Time
	ExportToClassic         bool
	ExportToENS             bool
}

// Config is a user-facing set of confuration data that every service should
// try to fill out
type Config struct {
	// Component should be the name of the system being instrumented. For clarity
	// and consistency this should be the same name as the Component Inventory
	// https://components.heroku.tools/components
	Component string
	// Environment is intended to help distinguish between "production", "staging"
	// and "test" environments, though there may be others
	Environment string
	// Version should be a string that changes every time you deploy your system.
	// This helps pin down when a particular problem was introduced or compare the
	// behavior of new versions as it rolls out.
	//
	// This can be a numeric string like "v123" which can be found on the platform
	// in the HEROKU_RELEASE_VERSION env var. The commit hash that was built and
	// deployed also works well.
	Version string
	// Instance should be an unique identifier "per machine". When using the Heroku
	// platform this would be found in the HEROKU_DYNO_ID env var.
	Instance string
	// Team should be included to help quickly identify who maintains a particular
	// system which can occasionally be tricky in incidents. For clarity and
	// consistency this should be the same name as used in the Component Inventory
	// https://components.heroku.tools/teams
	Team string
}

var (
	config = &TelemetryConfig{}
)

// WithConfig lets you specify ConfigOptions with initializing the telemetry-go library.
func WithConfig(c Config) ConfigOption {
	sdkMeta := sdk.GetMetadata(debug.ReadBuildInfo())

	return func(tc *TelemetryConfig) error {
		tc.UserConfig = c
		rsc, err := resource.Merge(tc.Resource, resource.NewSchemaless(
			semconv.DeploymentEnvironmentKey.String(c.Environment),
			attribute.String("service.component", c.Component),
			semconv.ServiceNameKey.String(c.Component),
			semconv.ServiceVersionKey.String(c.Version),
			semconv.HostNameKey.String(c.Instance),
			semconv.TelemetrySDKNameKey.String(sdkMeta.Name),
			semconv.TelemetrySDKLanguageKey.String(sdkMeta.Language),
			semconv.TelemetrySDKVersionKey.String(sdkMeta.Version),
			attribute.String("service.team", c.Team),
		))

		if err != nil {
			return err
		}

		tc.Resource = rsc
		return nil
	}
}

// WithMetricsEnabled will enable send metrics to Honeycomb.
func WithMetricsEnabled() ConfigOption {
	return func(tc *TelemetryConfig) error {
		tc.MetricsEnabled = true
		return nil
	}
}

// WithHoneycomb lets you specify the API Key and Dataset to use when relaying data to Honeycomb.
func WithHoneycomb(apiKey string, dataset string) ConfigOption {
	return func(tc *TelemetryConfig) error {
		tc.HoneycombAPIKey = apiKey
		tc.HoneycombTracesDataset = dataset
		config.TelemetryAddonEnabled = false
		return nil
	}
}

// WithHoneycombApiKey lets you specify the API Key to use when relaying data to Honeycomb.
func WithHoneycombApiKey(apiKey string) ConfigOption {
	return func(tc *TelemetryConfig) error {
		tc.HoneycombAPIKey = apiKey
		config.TelemetryAddonEnabled = false
		return nil
	}
}

// WithMetricsDatasetHoneycomb lets you specify the Metrics Dataset to use when relaying data to Honeycomb.
func WithMetricsDatasetHoneycomb(dataset string) ConfigOption {
	return func(tc *TelemetryConfig) error {
		tc.HoneycombMetricsDataset = dataset
		return nil
	}
}

// WithTracesDatasetHoneycomb lets you specify the Traces Dataset to use when relaying data to Honeycomb.
func WithTracesDatasetHoneycomb(dataset string) ConfigOption {
	return func(tc *TelemetryConfig) error {
		tc.HoneycombTracesDataset = dataset
		return nil
	}
}

// WithDiskRoot specifies the directory root config option.
func WithDiskRoot(root string) ConfigOption {
	return func(tc *TelemetryConfig) error {
		tc.DiskRoot = root
		return nil
	}
}

// WithTestExportedr lets you specify your own exporter for testing purposes.
// Sampling is turned off by default when testing.
func WithTestExporter(exporter *TestExporter) ConfigOption {
	return func(tc *TelemetryConfig) error {
		tc.TestExporter = exporter
		tc.SamplingRate = defaultSamplingRate
		return nil
	}
}

// WithLogWriter lets yous specify your own log writer.
func WithLogWriter(writer io.Writer) ConfigOption {
	return func(tc *TelemetryConfig) error {
		tc.LogWriter = writer
		return nil
	}
}

// WithGlobalAttributes lets you specify the key/value attributes that should be on every trace.
func WithGlobalAttributes(attrs ...attribute.KeyValue) ConfigOption {
	return func(tc *TelemetryConfig) error {
		rsc, err := resource.Merge(tc.Resource, resource.NewSchemaless(
			attrs...,
		))
		if err != nil {
			return err
		}
		tc.Resource = rsc
		return nil
	}
}

// WithSamplingRate lets you override the default sampling rate.
func WithSamplingRate(rate float64) ConfigOption {
	return func(bc *TelemetryConfig) error {
		bc.SamplingRate = rate
		return nil
	}
}

// WithVerboseSamplingRate lets you override the default verbose sampling rate.
func WithVerboseSamplingRate(rate float64) ConfigOption {
	return func(bc *TelemetryConfig) error {
		bc.VerboseSamplingRate = rate
		return nil
	}
}

// WithFilter allows a function to be configured that will
// be called on every request. If the filter function returns
// false that request will not be traced. Note that this may
// be the opposite of what you expect.
//
// This is helpful to avoid tracing requests like health
// checks that are of relatively little utility. To exclude
// only the /health route you could pass the following
// configuration:
//
//	telemetry.WithFilter(func(r *http.Request) bool {
//	 	if r != nil {
//	 		return r.URL.Path != "/health"
//	 	}
//	 	return true
//	}),
func WithFilter(filter otelhttp.Filter) ConfigOption {
	return func(bc *TelemetryConfig) error {
		bc.Filter = filter
		return nil
	}
}

// Configure configures all the observability components required.
// Defaults include sampling rate, verbose sampling rate, writing logs to STDOUT,
// Heroku dyno metadata as global attributes (see https://devcenter.heroku.com/articles/dyno-metadata),
// and more.
func Configure(opts ...ConfigOption) error {
	// Set a default to sampling rate for the library user to override
	config.SamplingRate = defaultSamplingRate
	config.VerboseSamplingRate = defaultVerboseSamplingRate
	config.LogWriter = os.Stdout
	config.StartTime = time.Now()
	// Set a default to disk root for the library user to override
	config.DiskRoot = defaultDiskRootValue
	h, err := getHerokuAppMetadata()

	if err == nil {
		config.Resource, err = processHerokuAppMetadata(h)
		if err != nil {
			return err
		}
	}

	// default honeycomb api key which comes with the herokai-telemetry addon
	apiKey := os.Getenv(defaultHoneycombApiKey)
	if len(apiKey) > 0 {
		config.HoneycombAPIKey = apiKey
		config.TelemetryAddonEnabled = true
	}

	for _, opt := range opts {
		err = opt(config)
		if err != nil {
			return err
		}
	}

	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(config.SamplingRate))
	limits := setupSpanLimits()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithRawSpanLimits(limits),
		sdktrace.WithResource(config.Resource),
	)
	// Store a reference to the TracerProvider we have configured in case the global
	// TracerProvider gets overwritten
	config.TracerProvider = provider

	// Set the global TracerProvider for convenience
	otel.SetTracerProvider(provider)

	registerLogExporter(config)
	registerTestExporter(config)

	honeycombAPIKey := config.HoneycombAPIKey
	if !keysAreValid(honeycombAPIKey) {
		fmt.Println("Invalid Honeycomb API Key format. Honeycomb keys must be valid and and distinct(at most 1 classic key and 1 E&S key).")
	}

	if keysAreValid(honeycombAPIKey) {
		keys := strings.Split(honeycombAPIKey, ";")

		for _, key := range keys {
			keyType := determineKeyType(key)

			if keyType == "CLASSIC" && config.HoneycombTracesDataset != "" {
				if err = registerHoneycombClassicTraceExporter(config, key); err != nil {
					return err
				}
			} else if keyType == "E_AND_S" {
				if err = registerHoneycombTraceExporter(key); err != nil {
					return err
				}
			}
		}
	}

	if keysAreValid(honeycombAPIKey) && config.MetricsEnabled && config.HoneycombMetricsDataset != "" {
		keys := strings.Split(honeycombAPIKey, ";")

		for _, key := range keys {
			if err = registerHoneycombMetricExporter(config, key); err != nil {
				return err
			}
		}
	}

	if err = reportUsage(config); err != nil {
		return err
	}

	return nil
}

func setupSpanLimits() sdktrace.SpanLimits {
	limits := sdktrace.NewSpanLimits()
	limits.AttributeCountLimit = maxAttributesPerSpan
	return limits
}

func reportUsage(tc *TelemetryConfig) error {
	if keysAreValid(tc.HoneycombAPIKey) {
		url := fmt.Sprint("https://api.honeycomb.io/1/events/", reportingDataset)
		body, err := buildUsageReport(tc)
		if err != nil {
			return err
		}

		apiKeys := strings.Split(tc.HoneycombAPIKey, ";")

		for _, key := range apiKeys {
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
			if err != nil {
				return err
			}

			req.Header.Add("X-Honeycomb-Team", key)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}

			defer resp.Body.Close()
		}
	}

	return nil
}

type usageReport struct {
	Component             string `json:"component"`
	TelemetryAddonEnabled bool   `json:"telemetry_addon_enabled"`
	TracesDataSet         string `json:"traces_dataset"`
	MetricsDataSet        string `json:"metrics_dataset"`
	Environment           string `json:"environment"`
	Version               string `json:"version"`
	Instance              string `json:"instance"`
	Team                  string `json:"team"`
	GoVersion             string `json:"go_version"`
	SdkVersion            string `json:"sdk_version"`
	SdkName               string `json:"sdk_name"`
	SdkLanguage           string `json:"sdk_language"`
	ExportToENS           bool   `json:"exports_to_ens"`
	ExportToClassic       bool   `json:"exports_to_classic"`
}

func buildUsageReport(tc *TelemetryConfig) ([]byte, error) {
	sdkMeta := sdk.GetMetadata(debug.ReadBuildInfo())

	report := usageReport{
		Component:             tc.UserConfig.Component,
		TelemetryAddonEnabled: tc.TelemetryAddonEnabled,
		TracesDataSet:         tc.HoneycombTracesDataset,
		MetricsDataSet:        tc.HoneycombMetricsDataset,
		Environment:           tc.UserConfig.Environment,
		Version:               tc.UserConfig.Version,
		Instance:              tc.UserConfig.Instance,
		Team:                  tc.UserConfig.Team,
		GoVersion:             runtime.Version(),
		SdkVersion:            sdkMeta.Version,
		SdkName:               sdkMeta.Name,
		SdkLanguage:           sdkMeta.Language,
		ExportToENS:           tc.ExportToENS,
		ExportToClassic:       tc.ExportToClassic,
	}

	return json.Marshal(report)
}

// Close closes all exporters for shutdown.
func Close() {
	_ = TracerProvider().Shutdown(context.Background())

	if config.MetricsEnabled {
		_ = MeterProvider().Shutdown(context.Background())
	}
	// reset the configuration to empty
	config = &TelemetryConfig{}
}

func registerLogExporter(tc *TelemetryConfig) {
	if tc.LogWriter != nil {
		exporter := &LogExporter{writer: tc.LogWriter}
		TracerProvider().RegisterSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter))
	}
}

func registerHoneycombTraceExporter(apiKey string) error {
	config.ExportToENS = true
	ctx := context.Background()
	exporter, err := otlptrace.New(
		ctx,
		otlptracegrpc.NewClient(
			otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")),
			otlptracegrpc.WithEndpoint("api.honeycomb.io:443"),
			otlptracegrpc.WithHeaders(map[string]string{
				"x-honeycomb-team": apiKey,
			}),
		),
	)
	if err != nil {
		return err
	}

	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	TracerProvider().RegisterSpanProcessor(bsp)

	return nil
}

func registerHoneycombClassicTraceExporter(tc *TelemetryConfig, apiKey string) error {
	config.ExportToClassic = true
	ctx := context.Background()
	exporter, err := otlptrace.New(
		ctx,
		otlptracegrpc.NewClient(
			otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")),
			otlptracegrpc.WithEndpoint("api.honeycomb.io:443"),
			otlptracegrpc.WithHeaders(map[string]string{
				"x-honeycomb-team":    apiKey,
				"x-honeycomb-dataset": tc.HoneycombTracesDataset,
			}),
		),
	)
	if err != nil {
		return err
	}

	// rewrite service.name
	component := tc.Resource.Attributes()[2].Value.AsString()
	environment := tc.Resource.Attributes()[0].Value.AsString()
	overwriteName := fmt.Sprintf("%s - %s", component, environment)

	customProcessor := newOverwriteServiceNameSpanProcessor(exporter, overwriteName)
	TracerProvider().RegisterSpanProcessor(customProcessor)

	return nil
}

func registerHoneycombMetricExporter(tc *TelemetryConfig, apiKey string) error {
	ctx := context.Background()
	exporter, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")),
		otlpmetricgrpc.WithEndpoint("api.honeycomb.io:443"),
		otlpmetricgrpc.WithHeaders(map[string]string{
			"x-honeycomb-team":    apiKey,
			"x-honeycomb-dataset": tc.HoneycombMetricsDataset,
		}),
		otlpmetricgrpc.WithTemporalitySelector(func(kind metric.InstrumentKind) metricdata.Temporality {
			return metricdata.DeltaTemporality
		}),
	)
	if err != nil {
		return err
	}

	tc.MeterProvider = metric.NewMeterProvider(
		metric.WithReader(
			metric.NewPeriodicReader(
				exporter,
				metric.WithInterval(1*time.Minute),
			)),
		metric.WithResource(config.Resource))
	otel.SetMeterProvider(tc.MeterProvider)

	tc.Meter = tc.MeterProvider.Meter(MeterName)

	return nil
}

func registerTestExporter(tc *TelemetryConfig) {
	if tc.TestExporter != nil {
		TracerProvider().RegisterSpanProcessor(sdktrace.NewSimpleSpanProcessor(tc.TestExporter))

		tc.MeterProvider = metric.NewMeterProvider(
			metric.WithReader(
				metric.NewPeriodicReader(
					tc.TestExporter,
					metric.WithInterval(200*time.Millisecond),
				)),
			metric.WithResource(config.Resource))

		otel.SetMeterProvider(tc.MeterProvider)

		tc.Meter = tc.MeterProvider.Meter(MeterName)
	}
}

func getHerokuAppMetadata() (HerokuAppMetadata, error) {
	h := HerokuAppMetadata{}
	err := envdecode.Decode(&h)

	// For our use-case, not decoding any env vars is not an error
	if errors.Is(envdecode.ErrNoTargetFieldsAreSet, err) {
		err = nil
	}

	// Dyno should be formatted like: web.1
	separated := strings.Split(h.Dyno, ".")
	if len(separated) == 2 {
		h.DynoType = separated[0]
		val, err := strconv.Atoi(separated[1])

		if err == nil {
			h.DynoIndex = val
		}
	}

	return h, err
}

func processHerokuAppMetadata(h HerokuAppMetadata) (*resource.Resource, error) {
	if h.Dyno == "" {
		return config.Resource, nil
	}
	if h.ReleaseVersion == "" {
		return resource.Merge(config.Resource, resource.NewSchemaless(
			attribute.String("heroku.dyno", h.Dyno),
			attribute.String("heroku.dyno_type", h.DynoType),
			attribute.Int("heroku.dyno_index", h.DynoIndex),
		))
	}
	return resource.Merge(config.Resource, resource.NewSchemaless(
		attribute.String("heroku.app_id", h.AppID),
		attribute.String("heroku.dyno_id", h.DynoID),
		attribute.String("heroku.dyno", h.Dyno),
		attribute.String("heroku.dyno_type", h.DynoType),
		attribute.Int("heroku.dyno_index", h.DynoIndex),
		attribute.String("heroku.app_name", h.AppName),
		attribute.String("heroku.release_created_at", h.ReleaseCreatedAt),
		attribute.String("heroku.slug_commit", h.SlugCommit),
		attribute.String("heroku.slug_description", h.SlugDescription),
		attribute.String("heroku.release_version", h.ReleaseVersion),
		semconv.ServiceVersionKey.String(h.ReleaseVersion),
	))
}
