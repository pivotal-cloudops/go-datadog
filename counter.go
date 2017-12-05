package datadog

// Inspired by https://github.com/rcrowley/go-metrics
// Copyright 2012 Richard Crowley. All rights reserved.

import "sync/atomic"

// Counter is the standard implementation of a Counter and uses the
// sync/atomic package to manage a single int64 value.
type Counter struct {
	BaseMetric
	count int64
}

// NewCounter creates a new counter
func NewCounter(name string, tags ...string) *Counter {
	return &Counter{BaseMetric: BaseMetric{name: name, tags: tags}}
}

// FetchCounter returns or registers a new one
func FetchCounter(rep *MetricReporter, name string, tags ...string) *Counter {
	return rep.Fetch(func() Metric { return NewCounter(name, tags...) }, name, tags...).(*Counter)
}

// RegisterCounter registers a counter
func RegisterCounter(rep *MetricReporter, name string, tags ...string) *Counter {
	m := NewCounter(name, tags...)
	rep.Register(m)
	return m
}

// Clear sets the counter to zero.
func (c *Counter) Clear() {
	atomic.StoreInt64(&c.count, 0)
}

// Count returns the current count.
func (c *Counter) Count() int64 {
	return atomic.LoadInt64(&c.count)
}

// Dec decrements the counter by the given amount.
func (c *Counter) Dec(i int64) {
	atomic.AddInt64(&c.count, -i)
}

// Inc increments the counter by the given amount.
func (c *Counter) Inc(i int64) {
	atomic.AddInt64(&c.count, i)
}

// Flush returns series
func (m *Counter) Flush(now int64) []*Series {
	return []*Series{
		NewSeries(m.name+".count", now, m.Count(), m.tags, MT_COUNTER),
	}
}

// FlashCounter is the a counter that resets to 0 after each flush
type FlashCounter struct {
	Counter
}

// NewFlashCounter creates a new reset counter
func NewFlashCounter(name string, tags ...string) *FlashCounter {
	return &FlashCounter{*NewCounter(name, tags...)}
}

// FetchFlashCounter returns or registers a new one
func FetchFlashCounter(rep *MetricReporter, name string, tags ...string) *FlashCounter {
	return rep.Fetch(func() Metric { return NewFlashCounter(name, tags...) }, name, tags...).(*FlashCounter)
}

// RegisterFlashCounter registers a reset counter
func RegisterFlashCounter(rep *MetricReporter, name string, tags ...string) *FlashCounter {
	m := NewFlashCounter(name, tags...)
	rep.Register(m)
	return m
}

// Flush returns series and resets counter
func (m *FlashCounter) Flush(now int64) []*Series {
	count := m.Count()
	defer m.Dec(count)

	return []*Series{
		NewSeries(m.name+".count", now, count, m.tags, MT_COUNTER),
	}
}
