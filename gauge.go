package datadog

import (
	"sync"
	"sync/atomic"
)

// Gauge is the standard implementation of a Gauge and uses the
// sync/atomic package to manage a single int64 value.
type Gauge struct {
	BaseMetric
	value int64
}

// NewGauge creates a new gauge
func NewGauge(name string, tags ...string) *Gauge {
	return &Gauge{BaseMetric: BaseMetric{name: name, tags: tags}}
}

// FetchGauge returns or registers a new one
func FetchGauge(rep *MetricReporter, name string, tags ...string) *Gauge {
	return rep.Fetch(func() Metric { return NewGauge(name, tags...) }, name, tags...).(*Gauge)
}

// RegisterGauge registers a gauge
func RegisterGauge(rep *MetricReporter, name string, tags ...string) *Gauge {
	m := NewGauge(name, tags...)
	rep.Register(m)
	return m
}

// Update updates the gauge's value.
func (g *Gauge) Update(v int64) {
	atomic.StoreInt64(&g.value, v)
}

// Value returns the gauge's current value.
func (g *Gauge) Value() int64 {
	return atomic.LoadInt64(&g.value)
}

// Flush returns series
func (m *Gauge) Flush(now int64) []*Series {
	return []*Series{
		NewSeries(m.name+".value", now, m.Value(), m.tags, MT_GAUGE),
	}
}

// GaugeF is like a normal Gauge, but holds floating point values
type GaugeF struct {
	BaseMetric
	value float64
	lock  sync.Mutex
}

// NewGaugeF creates a new gauge
func NewGaugeF(name string, tags ...string) *GaugeF {
	return &GaugeF{BaseMetric: BaseMetric{name: name, tags: tags}}
}

// FetchGaugeF returns or registers a new one
func FetchGaugeF(rep *MetricReporter, name string, tags ...string) *GaugeF {
	return rep.Fetch(func() Metric { return NewGaugeF(name, tags...) }, name, tags...).(*GaugeF)
}

// RegisterGauge (finds or) registers a gauge
func RegisterGaugeF(rep *MetricReporter, name string, tags ...string) *GaugeF {
	m := NewGaugeF(name, tags...)
	rep.Register(m)
	return m
}

// Update updates the gauge's value.
func (g *GaugeF) Update(v float64) {
	g.lock.Lock()
	g.value = v
	g.lock.Unlock()
}

// Value returns the gauge's current value.
func (g *GaugeF) Value() float64 {
	g.lock.Lock()
	v := g.value
	g.lock.Unlock()
	return v
}

// Flush returns series
func (m *GaugeF) Flush(now int64) []*Series {
	return []*Series{
		NewSeries(m.name+".value", now, m.Value(), m.tags, MT_GAUGE),
	}
}
