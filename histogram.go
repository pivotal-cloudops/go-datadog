package datadog

// A standard histogram
type Histogram struct {
	BaseMetric
	sample Sample
}

// NewCustomHistogram creates a new custom histogram
func NewCustomHistogram(name string, sample Sample, tags ...string) *Histogram {
	return &Histogram{BaseMetric: BaseMetric{name: name, tags: tags}, sample: sample}
}

// FetchCustomHistogram returns or registers a new one
func FetchCustomHistogram(rep *MetricReporter, name string, sample Sample, tags ...string) *Histogram {
	return rep.Fetch(func() Metric { return NewCustomHistogram(name, sample, tags...) }, name, tags...).(*Histogram)
}

// RegisterCustomHistogram registers a histogram
func RegisterCustomHistogram(rep *MetricReporter, name string, sample Sample, tags ...string) *Histogram {
	m := NewCustomHistogram(name, sample, tags...)
	rep.Register(m)
	return m
}

// NewHistogram creates a new histogram with default sampling
func NewHistogram(name string, tags ...string) *Histogram {
	return NewCustomHistogram(name, NewDefaultSample(), tags...)
}

// FetchHistogram returns or registers a new one
func FetchHistogram(rep *MetricReporter, name string, tags ...string) *Histogram {
	return rep.Fetch(func() Metric { return NewHistogram(name, tags...) }, name, tags...).(*Histogram)
}

// RegisterHistogram registers a histogram
func RegisterHistogram(rep *MetricReporter, name string, tags ...string) *Histogram {
	return RegisterCustomHistogram(rep, name, NewDefaultSample(), tags...)
}

// Clear clears the histogram and its sample.
func (h *Histogram) Clear() { h.sample.Clear() }

// Snapshot returns a read-only snapshot for statistical analysis
func (h *Histogram) Snapshot() *SampleSnapshot { return h.sample.Snapshot() }

// Update samples a new value.
func (h *Histogram) Update(v int64) { h.sample.Update(v) }

// Flush returns series
func (h *Histogram) Flush(now int64) []*Series {
	snap := h.Snapshot()
	p := snap.Percentiles([]float64{0.5, 0.75, 0.95, 0.99})
	return []*Series{
		NewSeries(h.name+".count", now, snap.Count(), h.tags, MT_COUNTER),
		NewSeries(h.name+".min", now, snap.Min(), h.tags, MT_GAUGE),
		NewSeries(h.name+".max", now, snap.Max(), h.tags, MT_GAUGE),
		NewSeries(h.name+".mean", now, snap.Mean(), h.tags, MT_GAUGE),
		NewSeries(h.name+".stddev", now, snap.StdDev(), h.tags, MT_GAUGE),
		NewSeries(h.name+".median", now, p[0], h.tags, MT_GAUGE),
		NewSeries(h.name+".percentile.75", now, p[1], h.tags, MT_GAUGE),
		NewSeries(h.name+".percentile.95", now, p[2], h.tags, MT_GAUGE),
		NewSeries(h.name+".percentile.99", now, p[3], h.tags, MT_GAUGE),
	}
}
