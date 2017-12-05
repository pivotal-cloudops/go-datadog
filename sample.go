package datadog

// Taken from https://github.com/rcrowley/go-metrics
// Copyright 2012 Richard Crowley. All rights reserved.

import (
	"container/heap"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"
)

const rescaleThreshold = time.Hour

// NewDefaultSample is a default constructor using an exponentially-decaying
// sample with the same reservoir size and alpha as UNIX load averages.
func NewDefaultSample() Sample { return NewExpDecaySample(1028, 0.015) }

// Samples maintain a statistically-significant selection of values from
// a stream.
type Sample interface {
	Clear()
	Update(int64)
	Snapshot() *SampleSnapshot
	Size() int
	Count() int64
	Values() []int64
}

// SampleSnapshot is a read-only copy of another Sample.
type SampleSnapshot struct {
	count  int64
	values int64Slice
}

// NewSampleSnapshot creates a new snapshot instance
func NewSampleSnapshot(count int64, values []int64) *SampleSnapshot {
	return &SampleSnapshot{count, values}
}

// Count returns the count of inputs at the time the snapshot was taken.
func (s *SampleSnapshot) Count() int64 { return s.count }

// Max returns the maximal value at the time the snapshot was taken.
func (s *SampleSnapshot) Max() int64 {
	if 0 == len(s.values) {
		return 0
	}
	var max int64 = math.MinInt64
	for _, v := range s.values {
		if max < v {
			max = v
		}
	}
	return max
}

// Mean returns the mean value at the time the snapshot was taken.
func (s *SampleSnapshot) Mean() float64 {
	if 0 == len(s.values) {
		return 0.0
	}
	return float64(s.Sum()) / float64(len(s.values))
}

// Min returns the minimal value at the time the snapshot was taken.
func (s *SampleSnapshot) Min() int64 {
	if 0 == len(s.values) {
		return 0
	}
	var min int64 = math.MaxInt64
	for _, v := range s.values {
		if min > v {
			min = v
		}
	}
	return min
}

// Percentile returns an arbitrary percentile of values at the time the
// snapshot was taken.
func (s *SampleSnapshot) Percentile(p float64) float64 {
	if len(s.values) == 0 {
		return 0.0
	}
	return s.Percentiles([]float64{p})[0]
}

// Percentiles returns a slice of arbitrary percentiles of values at the time
// the snapshot was taken.
func (s *SampleSnapshot) Percentiles(ps []float64) []float64 {
	scores := make([]float64, len(ps))

	if size := len(s.values); size > 0 {
		sort.Sort(s.values)
		for i, p := range ps {
			pos := p * float64(size+1)
			if pos < 1.0 {
				scores[i] = float64(s.values[0])
			} else if pos >= float64(size) {
				scores[i] = float64(s.values[size-1])
			} else {
				lower := float64(s.values[int(pos)-1])
				upper := float64(s.values[int(pos)])
				scores[i] = lower + (pos-math.Floor(pos))*(upper-lower)
			}
		}
	}
	return scores
}

// Size returns the size of the sample at the time the snapshot was taken.
func (s *SampleSnapshot) Size() int { return len(s.values) }

// StdDev returns the standard deviation of values at the time the snapshot was
// taken.
func (s *SampleSnapshot) StdDev() float64 { return math.Sqrt(s.Variance()) }

// Sum returns the sum of values at the time the snapshot was taken.
func (s *SampleSnapshot) Sum() int64 {
	var sum int64
	for _, v := range s.values {
		sum += v
	}
	return sum
}

// Variance returns the variance of values at the time the snapshot was taken.
func (s *SampleSnapshot) Variance() float64 {
	size := len(s.values)
	if size == 0 {
		return 0.0
	}
	m := s.Mean()
	var sum float64
	for _, v := range s.values {
		d := float64(v) - m
		sum += d * d
	}
	return sum / float64(size)
}

// ExpDecaySample is an exponentially-decaying sample using a forward-decaying
// priority reservoir.  See Cormode et al's "Forward Decay: A Practical Time
// Decay Model for Streaming Systems".
//
// <http://www.research.att.com/people/Cormode_Graham/library/publications/CormodeShkapenyukSrivastavaXu09.pdf>
type ExpDecaySample struct {
	alpha         float64
	count         int64
	mutex         sync.Mutex
	reservoirSize int
	t0, t1        time.Time
	values        expDecaySampleHeap
}

// NewExpDecaySample constructs a new exponentially-decaying sample with the
// given reservoir size and alpha.
func NewExpDecaySample(reservoirSize int, alpha float64) *ExpDecaySample {
	s := &ExpDecaySample{
		alpha:         alpha,
		reservoirSize: reservoirSize,
		t0:            time.Now(),
		values:        make(expDecaySampleHeap, 0, reservoirSize),
	}
	s.t1 = time.Now().Add(rescaleThreshold)
	return s
}

// Snapshot creates a read-only snapshot for statistical analysis
func (s *ExpDecaySample) Snapshot() *SampleSnapshot { return NewSampleSnapshot(s.Count(), s.Values()) }

// Clear clears all samples.
func (s *ExpDecaySample) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.count = 0
	s.t0 = time.Now()
	s.t1 = s.t0.Add(rescaleThreshold)
	s.values = make(expDecaySampleHeap, 0, s.reservoirSize)
}

// Count returns the number of samples recorded, which may exceed the
// reservoir size.
func (s *ExpDecaySample) Count() int64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.count
}

// Size returns the size of the sample, which is at most the reservoir size.
func (s *ExpDecaySample) Size() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return len(s.values)
}

// Update samples a new value.
func (s *ExpDecaySample) Update(v int64) {
	s.update(time.Now(), v)
}

// Values returns a copy of the values in the sample.
func (s *ExpDecaySample) Values() []int64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	values := make([]int64, len(s.values))
	for i, v := range s.values {
		values[i] = v.v
	}
	return values
}

// update samples a new value at a particular timestamp.  This is a method all
// its own to facilitate testing.
func (s *ExpDecaySample) update(t time.Time, v int64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.count++
	if len(s.values) == s.reservoirSize {
		heap.Pop(&s.values)
	}
	heap.Push(&s.values, expDecaySample{
		k: math.Exp(t.Sub(s.t0).Seconds()*s.alpha) / rand.Float64(),
		v: v,
	})
	if t.After(s.t1) {
		values := s.values
		t0 := s.t0
		s.values = make(expDecaySampleHeap, 0, s.reservoirSize)
		s.t0 = t
		s.t1 = s.t0.Add(rescaleThreshold)
		for _, v := range values {
			v.k = v.k * math.Exp(-s.alpha*float64(s.t0.Sub(t0)))
			heap.Push(&s.values, v)
		}
	}
}

// A uniform sample using Vitter's Algorithm R.
//
// <http://www.cs.umd.edu/~samir/498/vitter.pdf>
type UniformSample struct {
	count         int64
	mutex         sync.Mutex
	reservoirSize int
	values        []int64
}

// NewUniformSample constructs a new uniform sample with the given reservoir
// size.
func NewUniformSample(reservoirSize int) *UniformSample {
	return &UniformSample{
		reservoirSize: reservoirSize,
		values:        make([]int64, 0, reservoirSize),
	}
}

// Snapshot creates a read-only snapshot for statistical analysis
func (s *UniformSample) Snapshot() *SampleSnapshot { return NewSampleSnapshot(s.Count(), s.Values()) }

// Clear clears all samples.
func (s *UniformSample) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.count = 0
	s.values = make([]int64, 0, s.reservoirSize)
}

// Count returns the number of samples recorded, which may exceed the
// reservoir size.
func (s *UniformSample) Count() int64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.count
}

// Size returns the size of the sample, which is at most the reservoir size.
func (s *UniformSample) Size() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return len(s.values)
}

// Update samples a new value.
func (s *UniformSample) Update(v int64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.count++
	if len(s.values) < s.reservoirSize {
		s.values = append(s.values, v)
	} else {
		s.values[rand.Intn(s.reservoirSize)] = v
	}
}

// Values returns a copy of the values in the sample.
func (s *UniformSample) Values() []int64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	values := make([]int64, len(s.values))
	copy(values, s.values)
	return values
}

// A uniform sample which is cleared on every snapshot request
type FlashSample struct{ UniformSample }

// NewFlashSample constructs a new uniform flash sample with the given
// reservoir size.
func NewFlashSample(reservoirSize int) *FlashSample {
	return &FlashSample{*NewUniformSample(reservoirSize)}
}

// Snapshot creates a read-only snapshot for statistical analysis
func (s *FlashSample) Snapshot() *SampleSnapshot {
	snap := s.UniformSample.Snapshot()
	s.Clear()
	return snap
}

// expDecaySample represents an individual sample in a heap.
type expDecaySample struct {
	k float64
	v int64
}

// expDecaySampleHeap is a min-heap of expDecaySamples.
type expDecaySampleHeap []expDecaySample

func (q expDecaySampleHeap) Len() int {
	return len(q)
}

func (q expDecaySampleHeap) Less(i, j int) bool {
	return q[i].k < q[j].k
}

func (q *expDecaySampleHeap) Pop() interface{} {
	q_ := *q
	n := len(q_)
	i := q_[n-1]
	q_ = q_[0 : n-1]
	*q = q_
	return i
}

func (q *expDecaySampleHeap) Push(x interface{}) {
	q_ := *q
	n := len(q_)
	q_ = q_[0 : n+1]
	q_[n] = x.(expDecaySample)
	*q = q_
}

func (q expDecaySampleHeap) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
}

type int64Slice []int64

func (p int64Slice) Len() int           { return len(p) }
func (p int64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p int64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
