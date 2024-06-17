package sdk

import "runtime/debug"

// The import path for the telemetry-go library.
const TelemetryGoPackagePath string = "github.com/heroku/telemetry-go"

// Metadata holds metadata about the heroku/telemetry-go library.
type Metadata struct {
	Name     string
	Language string
	Version  string
}

// GetMetadata relies on runtime build information to primarily determine the version of the telemetry-go library being used by an executable.
// The ignored boolean value is a side effect of making it easier to pass in the result of a runtime/debug.ReadBuildInfo() call.
func GetMetadata(bi *debug.BuildInfo, _ bool) Metadata {
	// The default should have its Version overriden.
	md := Metadata{
		Name:     TelemetryGoPackagePath,
		Language: "go",
		Version:  "development",
	}

	if bi == nil {
		return md
	}

	for _, mod := range bi.Deps {
		if mod.Path == md.Name {
			md.Version = mod.Version
			break
		}
	}

	return md
}
