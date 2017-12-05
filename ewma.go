package datadog

// Taken from https://github.com/rcrowley/go-metrics
// Copyright 2012 Richard Crowley. All rights reserved.

import (
	"math"
	"sync"
	"sync/atomic"
)

// NewEWMA constructs a new EWMA with the given alpha.
func NewEWMA(alpha float64) *EWMA {
	return &EWMA{alpha: alpha}
}

// NewEWMA1 constructs a new EWMA for a one-minute moving average.
func NewEWMA1() *EWMA {
	return NewEWMA(1 - math.Exp(-5.0/60.0/1))
}

// NewEWMA5 constructs a new EWMA for a five-minute moving average.
func NewEWMA5() *EWMA {
	return NewEWMA(1 - math.Exp(-5.0/60.0/5))
}

// NewEWMA15 constructs a new EWMA for a fifteen-minute moving average.
func NewEWMA15() *EWMA {
	return NewEWMA(1 - math.Exp(-5.0/60.0/15))
}

// EWMA is the standard implementation of an EWMA and tracks the number
// of uncounted events and processes them on each tick.  It uses the
// sync/atomic package to manage uncounted events.
type EWMA struct {
	uncounted int64 // /!\ this should be the first member to ensure 64-bit alignment
	alpha     float64
	rate      float64
	init      bool
	mutex     sync.Mutex
}

// Rate returns the moving average rate of events per second.
func (a *EWMA) Rate() float64 {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.rate * float64(1e9)
}

// Tick ticks the clock to update the moving average.  It assumes it is called
// every five seconds.
func (a *EWMA) Tick() {
	count := atomic.LoadInt64(&a.uncounted)
	atomic.AddInt64(&a.uncounted, -count)
	instantRate := float64(count) / float64(5e9)
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if a.init {
		a.rate += a.alpha * (instantRate - a.rate)
	} else {
		a.init = true
		a.rate = instantRate
	}
}

// Update adds n uncounted events.
func (a *EWMA) Update(n int64) {
	atomic.AddInt64(&a.uncounted, n)
}
