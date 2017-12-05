package datadog

import "time"

// A standard timer
type Timer struct {
	*Meter
	unit   float64
	sample Sample
}

// NewCustomTimer creates a new timer
func NewCustomTimer(name string, unit time.Duration, sample Sample, tags ...string) *Timer {
	return &Timer{NewMeter(name, tags...), float64(unit), sample}
}

// FetchCustomTimer returns or registers a new one
func FetchCustomTimer(rep *MetricReporter, name string, unit time.Duration, sample Sample, tags ...string) *Timer {
	return rep.Fetch(func() Metric { return NewCustomTimer(name, unit, sample, tags...) }, name, tags...).(*Timer)
}

// RegisterCustomTimer registers a meter
func RegisterCustomTimer(rep *MetricReporter, name string, unit time.Duration, sample Sample, tags ...string) *Timer {
	m := NewCustomTimer(name, unit, sample, tags...)
	rep.Register(m)
	return m
}

// NewTimer creates a new timer with a default exponentially-decaying sample
func NewTimer(name string, unit time.Duration, tags ...string) *Timer {
	return NewCustomTimer(name, unit, NewDefaultSample(), tags...)
}

// FetchTimer returns or registers a new one
func FetchTimer(rep *MetricReporter, name string, unit time.Duration, tags ...string) *Timer {
	return rep.Fetch(func() Metric { return NewTimer(name, unit, tags...) }, name, tags...).(*Timer)
}

// RegisterTimer registers a meter
func RegisterTimer(rep *MetricReporter, name string, unit time.Duration, tags ...string) *Timer {
	return RegisterCustomTimer(rep, name, unit, NewDefaultSample(), tags...)
}

// Clear clears the histogram and its sample.
func (t *Timer) Clear() { t.sample.Clear() }

// Snapshot returns a read-only snapshot for statistical analysis
func (t *Timer) Snapshot() *SampleSnapshot { return t.sample.Snapshot() }

// Update records the duration of an event.
func (t *Timer) Update(d time.Duration) {
	t.sample.Update(int64(d))
	t.Mark(1)
}

// UpdateSince records the duration of an event that started at a time and ends now.
func (t *Timer) UpdateSince(ts time.Time) { t.Update(time.Now().Sub(ts)) }

// Flush returns series
func (t *Timer) Flush(now int64) []*Series {
	snap := t.Snapshot()
	p := snap.Percentiles([]float64{0.5, 0.75, 0.95, 0.99})
	return []*Series{
		NewSeries(t.name+".rate", now, t.RateMean(), t.tags, MT_GAUGE),
		NewSeries(t.name+".rate1", now, t.Rate1(), t.tags, MT_GAUGE),
		NewSeries(t.name+".rate5", now, t.Rate5(), t.tags, MT_GAUGE),
		NewSeries(t.name+".rate15", now, t.Rate15(), t.tags, MT_GAUGE),
		NewSeries(t.name+".count", now, snap.Count(), t.tags, MT_COUNTER),
		NewSeries(t.name+".min", now, t.norm(snap.Min()), t.tags, MT_GAUGE),
		NewSeries(t.name+".max", now, t.norm(snap.Max()), t.tags, MT_GAUGE),
		NewSeries(t.name+".mean", now, snap.Mean()/t.unit, t.tags, MT_GAUGE),
		NewSeries(t.name+".stddev", now, snap.StdDev()/t.unit, t.tags, MT_GAUGE),
		NewSeries(t.name+".median", now, p[0]/t.unit, t.tags, MT_GAUGE),
		NewSeries(t.name+".percentile.75", now, p[1]/t.unit, t.tags, MT_GAUGE),
		NewSeries(t.name+".percentile.95", now, p[2]/t.unit, t.tags, MT_GAUGE),
		NewSeries(t.name+".percentile.99", now, p[3]/t.unit, t.tags, MT_GAUGE),
	}
}

func (t *Timer) norm(n int64) float64 { return float64(n) / t.unit }
