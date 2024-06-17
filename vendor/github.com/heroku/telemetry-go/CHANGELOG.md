# Changelog

All notable changes to this project will be documented in this file.

We are not cutting explicit releases for this project. The `main` branch is always expected to be stable.
Changelog entries are sorted by the month when they were released.

## December 2020

- Switch from OpenCensus to OpenTelemetry ([#17](https://github.com/heroku/telemetry-go/pull/17)).
- Add `error` return attribute to the `Configure()` and `ConfigureForTesting()` methods ([#16](https://github.com/heroku/telemetry-go/pull/16)).
- Remove NewRelic exporter ([#15](https://github.com/heroku/telemetry-go/pull/15)).

## November 2020

- Rename "root" span to "main" span ([#8](https://github.com/heroku/telemetry-go/pull/8)).
- Make DiskRoot config optional, defaulting to `/app` ([#5](https://github.com/heroku/telemetry-go/pull/5)).
- Initial version extracted from [endosome](https://github.com/heroku/endosome).
