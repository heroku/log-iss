/*
Package telemetry is the Go implementation of the Heroku/Evergreen Platform Observability configuration.
The library provides a set of integrations to allow easy generation of standardized telemetry from Heroku and Evergreen components.

# Configuration

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
*/
package telemetry
