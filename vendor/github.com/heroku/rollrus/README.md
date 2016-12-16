# What

Rollrus is what happens when [Logrus](https://github.com/Sirupsen/logrus) meets [Roll](https://github.com/stvp/roll).

When a .Error, .Fatal or .Panic logging function is called, report the details to rollbar via a Logrus hook.

Delivery is synchronous to help ensure that logs are delivered.

# Usage

```go
package main

import  (
  "os"

  log "github.com/Sirupsen/logrus"
  "github.com/heroku/rollrus"
)

func main() {
  rollrus.SetupLogging(os.Getenv("ROLLBAR_TOKEN"), os.Getenv("ENVIRONMENT"))

  # This is not reported to Rollbar
  log.Info("OHAI")

  # This is reported to Rollbar
  log.WithFields(log.Fields{"hi":"there"}).Fatal("The end.")
}
```
