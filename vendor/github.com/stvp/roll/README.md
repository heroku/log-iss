roll
----

`roll` is a simple Rollbar client for Go that makes it easy to report errors and
messages to Rollbar. It supports all Rollbar error and message types, stack
traces, and automatic grouping by error type and location. For more advanced
functionality, check out [heroku/rollbar](https://github.com/heroku/rollbar).

All errors and messages are sent to Rollbar synchronously.

[API docs on godoc.org](http://godoc.org/github.com/stvp/roll)

Notes
=====

* Critical-, Error-, and Warning-level messages include a stack trace. However,
  Go's `error` type doesn't include stack information from the location the
  error was set or allocated. Instead, `roll` uses the stack information from
  where the error was reported.
* Info- and Debug-level Rollbar messages do not include stack traces.

Running Tests
=============

Set up a dummy project in Rollbar and pass the access token as an environment
variable to `go test`:

    TOKEN=f0df01587b8f76b2c217af34c479f9ea go test

Verify the reported errors manually in the Rollbar dashboard.

Contributors
============

This library was forked from [stvp/rollbar](https://github.com/stvp/rollbar),
which had contributions from:

* @kjk
* @Soulou
* @paulmach

