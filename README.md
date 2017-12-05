# Go Datadog [![Build Status](https://travis-ci.org/bsm/go-datadog.png)](https://travis-ci.org/bsm/go-datadog)

Simple [Go](http://golang.org/) interface to the [Datadog
API](http://docs.datadoghq.com/api/).


## Usage

```go
import(
  "github.com/bsm/datadog"
  "os"
  "time"
)

// Create a new client
host, _ := os.Hostname()
client := datadog.New(host, "dog-api-key")

// Init a reporter with tags
reporter := client.Reporter("some:tag1", "other:tag2")

// Register metrics
counterX, _ := datadog.RegisterCounter(reporter, "page.visits", "page:x")
counterY, _ := datadog.RegisterCounter(reporter, "page.visits", "page:y")
gaugeMem, _ := datadog.RegisterGauge(reporter, "mem.free")

// Start reporing, every 60s
go reporter.Start(60 * time.Second())

// Start collecting data
gaugeMem.Update(100)
counterX.Inc(15)
counterY.Inc(15)
```

## Custom Metrics

This lib comes with a few pre-defined metric types, but you can create your own
custom metrics by implementing the [Metric](metric.go) interface. For detailed
examples please see:

* [Gauge](gauge.go)
* [Counter & RateCounter](counter.go)
* [Histogram](histogram.go)
