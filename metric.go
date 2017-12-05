package datadog

import (
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	MT_COUNTER = "counter"
	MT_GAUGE   = "gauge"
)

// An abstract meter
type Metric interface {
	// Name returns the name
	Name() string
	// Tags returns the tags
	Tags() []string
	// Flush flushes meter and returns series
	// Accepts the current unix timestamp
	Flush(int64) []*Series
}

// Abstract base metric
type BaseMetric struct {
	name string
	tags []string
}

func (m *BaseMetric) Name() string   { return m.name }
func (m *BaseMetric) Tags() []string { return m.tags }

// MetricID
type MetricID string

// NewMetricID generates a unique metric ID using name and tags
func NewMetricID(name string, tags []string) string {
	sort.Strings(tags)
	return name + "|" + strings.Join(tags, ",")
}

// Periodic metric arbiter
// Ticks metrics on the scheduled intervals

type tickableMetric interface {
	Metric
	tick()
}

type tickableArbiter struct {
	sync.Mutex
	started bool
	metrics []tickableMetric
}

var arbiter = new(tickableArbiter)

func (ta *tickableArbiter) loop() {
	ticker := time.NewTicker(5e9)
	for {
		select {
		case <-ticker.C:
			ta.Lock()
			for _, metric := range ta.metrics {
				metric.tick()
			}
			ta.Unlock()
		}
	}
}

func (ta *tickableArbiter) add(m tickableMetric) {
	ta.Lock()
	defer ta.Unlock()

	ta.metrics = append(ta.metrics, m)
	if !ta.started {
		ta.started = true
		go ta.loop()
	}
}
