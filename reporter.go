// Datadog reporter for the [go-metrics](https://github.com/rcrowley/go-metrics)
// library.
package datadog

import (
	"log"
	"sync"
	"time"
)

type MetricReporter struct {
	client   *Client
	registry map[string]Metric
	tags     []string
	lock     sync.Mutex
}

// NewReporter creates an un-started Reporter.
// The recreated `Reporter` will not be started. Invoke `go r.Start()`
// to enable reporting.
func NewReporter(c *Client, t ...string) *MetricReporter {
	return &MetricReporter{
		client:   c,
		tags:     t,
		registry: make(map[string]Metric),
	}
}

// Start this reporter in a blocking fashion, pushing series data to datadog at
// the specified interval. If any errors occur, they will be logged to the
// default logger, and further updates will continue.
//
// Scheduling is done with a `time.Ticker`, so non-overlapping intervals are
// absolute, not based on the finish time of the previous event. They are,
// however, serial.
func (rep *MetricReporter) Start(d time.Duration) {
	ticker := time.NewTicker(d)
	for _ = range ticker.C {
		if err := rep.Report(); err != nil {
			log.Printf("Datadog series error: %s", err.Error())
		}
	}
}

// Register registers a single metric
func (rep *MetricReporter) Register(m Metric) {
	rep.lock.Lock()
	rep.registry[NewMetricID(m.Name(), m.Tags())] = m
	rep.lock.Unlock()
}

// Get returns a registered metric
func (rep *MetricReporter) Get(name string, tags ...string) Metric {
	return rep.GetByID(NewMetricID(name, tags))
}

// Fetch returns a registered metric or registers a new one via given fallback
func (rep *MetricReporter) Fetch(fallback func() Metric, name string, tags ...string) Metric {
	id := NewMetricID(name, tags)

	rep.lock.Lock()
	defer rep.lock.Unlock()

	val, ok := rep.registry[id]
	if !ok {
		val = fallback()
		rep.registry[id] = val
	}
	return val
}

// GetByID returns a registered metric
func (rep *MetricReporter) GetByID(id string) Metric {
	rep.lock.Lock()
	val, ok := rep.registry[id]
	rep.lock.Unlock()

	if ok {
		return val
	}
	return nil
}

// Report POSTs a single series report to the Datadog API. A 200 or 202 is expected for
// this to complete without error.
func (rep *MetricReporter) Report() error {
	return rep.client.PostSeries(rep.Series())
}

// Series flushes each metric associated with the reporter and returns a series messages
// with the current hostname of the `Client`.
func (rep *MetricReporter) Series() []*Series {
	now := time.Now().Unix()
	mets := rep.registered()

	series := make([]*Series, 0, len(mets))
	for _, m := range mets {
		series = append(series, m.Flush(now)...)
	}

	for _, s := range series {
		s.Tags = append(s.Tags, rep.tags...)
		s.Host = rep.client.Host
	}

	return series
}

func (rep *MetricReporter) registered() []Metric {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	ms := make([]Metric, 0, len(rep.registry))
	for _, m := range rep.registry {
		ms = append(ms, m)
	}
	return ms
}
