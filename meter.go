package datadog

import (
	"sync"
	"sync/atomic"
	"time"
)

// NewMeter creates a new meter
func NewMeter(name string, tags ...string) *Meter {
	m := &Meter{
		BaseMetric: BaseMetric{name: name, tags: tags},
		a1:         NewEWMA1(),
		a5:         NewEWMA5(),
		a15:        NewEWMA15(),
		startTime:  time.Now(),
	}
	arbiter.add(m)
	return m
}

// FetchMeter returns or registers a new one
func FetchMeter(rep *MetricReporter, name string, tags ...string) *Meter {
	return rep.Fetch(func() Metric { return NewMeter(name, tags...) }, name, tags...).(*Meter)
}

// RegisterMeter registers a meter
func RegisterMeter(rep *MetricReporter, name string, tags ...string) *Meter {
	m := NewMeter(name, tags...)
	rep.Register(m)
	return m
}

// Meter is the standard implementation of a Meter.
type Meter struct {
	BaseMetric
	lock sync.Mutex

	count     int64
	startTime time.Time

	rate1, rate5, rate15, rateMean float64
	a1, a5, a15                    *EWMA
}

// Count returns the number of events recorded.
func (m *Meter) Count() int64 {
	return atomic.LoadInt64(&m.count)
}

// Mark records the occurance of n events.
func (m *Meter) Mark(n int64) {
	atomic.AddInt64(&m.count, n)
	m.a1.Update(n)
	m.a5.Update(n)
	m.a15.Update(n)
}

// Rate1 returns the one-minute moving average rate of events per second.
func (m *Meter) Rate1() float64 {
	m.lock.Lock()
	rate := m.rate1
	m.lock.Unlock()
	return rate
}

// Rate5 returns the five-minute moving average rate of events per second.
func (m *Meter) Rate5() float64 {
	m.lock.Lock()
	rate := m.rate5
	m.lock.Unlock()
	return rate
}

// Rate15 returns the fifteen-minute moving average rate of events per second.
func (m *Meter) Rate15() float64 {
	m.lock.Lock()
	rate := m.rate15
	m.lock.Unlock()
	return rate
}

// RateMean returns the meter's mean rate of events per second.
func (m *Meter) RateMean() float64 {
	m.lock.Lock()
	rateMean := m.rateMean
	m.lock.Unlock()
	return rateMean
}

func (m *Meter) tick() {
	m.a1.Tick()
	m.a5.Tick()
	m.a15.Tick()

	m.lock.Lock()
	defer m.lock.Unlock()

	m.rate1 = m.a1.Rate()
	m.rate5 = m.a5.Rate()
	m.rate15 = m.a15.Rate()
	m.rateMean = float64(m.Count()) / time.Since(m.startTime).Seconds()
}

// Flush returns series and resets counter
func (m *Meter) Flush(now int64) []*Series {
	return []*Series{
		NewSeries(m.name+".rate", now, m.RateMean(), m.tags, MT_GAUGE),
		NewSeries(m.name+".rate1", now, m.Rate1(), m.tags, MT_GAUGE),
		NewSeries(m.name+".rate5", now, m.Rate5(), m.tags, MT_GAUGE),
		NewSeries(m.name+".rate15", now, m.Rate15(), m.tags, MT_GAUGE),
	}
}
