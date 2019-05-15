# Estafette

The `estafette-foundation` library is part of the Estafette CI system documented at https://estafette.io.

Please file any issues related to Estafette CI at https://github.com/estafette/estafette-ci-central/issues

## Estafette-foundation

This library provides building blocks for creating

This library has contracts for requests / responses between various components of the Estafette CI system.

## Development

To start development run

```bash
git clone git@github.com:estafette/estafette-foundation.git
cd estafette-foundation
```

Before committing your changes run

```bash
go test ./...
go mod tidy
go mod vendor
```

## Usage

To add this module to your golang application run

```bash
go get github.com/estafette/estafette-foundation
```

### Initialize logging

```go
import "github.com/estafette/estafette-foundation"

foundation.InitLogging(app, version, branch, revision, buildDate)
```

### Initialize Prometheus metrics endpoint

```go
import "github.com/estafette/estafette-foundation"

foundation.InitMetrics()
```

### Handle graceful shutdown

```go
import "github.com/estafette/estafette-foundation"

gracefulShutdown, waitGroup := foundation.InitGracefulShutdownHandling()

// your core application logic, making use of the waitGroup for critical sections

foundation.HandleGracefulShutdown(gracefulShutdown, waitGroup)
```


### Watch mounted folder for changes

```go
import "github.com/estafette/estafette-foundation"

foundation.WatchForFileChanges("/path/to/mounted/secret/or/configmap", func(event fsnotify.Event) {
  // reinitialize parts making use of the mounted data
})
```

### Apply jitter to a number to introduce randomness

Inspired by http://highscalability.com/blog/2012/4/17/youtube-strategy-adding-jitter-isnt-a-bug.html you want to add jitter to a lot of parts of your platform, like cache durations, polling intervals, etc.

```go
import "github.com/estafette/estafette-foundation"

sleepTime := foundation.ApplyJitter(30)
time.Sleep(time.Duration(sleepTime) * time.Second)
```