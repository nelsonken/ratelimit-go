package ratelimit

import (
	"sync/atomic"
	"time"
	"unsafe"
)

type Limiter interface {
	// Take return can access or not immediately
	Take() bool

	// SpinTake spin while can't access with timeout
	SpinTake(timeout time.Duration) bool
}

func New(rate, maxBurst int, duration time.Duration) Limiter {
	return newAtomicBased(rate, maxBurst, duration)
}

type state struct {
	last     time.Time
	sleepFor time.Duration
}

// newAtomicBased returns a new atomic based limiter.
func newAtomicBased(rate, maxBurst int, duration time.Duration) *atomicLimiter {
	perRequest := duration / time.Duration(rate)
	l := &atomicLimiter{
		perRequest: perRequest,
		maxBurst:   -1 * time.Duration(maxBurst) * perRequest,
	}

	initialState := state{
		last:     time.Time{},
		sleepFor: 0,
	}

	atomic.StorePointer(&l.state, unsafe.Pointer(&initialState))
	return l
}

type atomicLimiter struct {
	state unsafe.Pointer
	//lint:ignore U1000 Padding is unused but it is crucial to maintain performance
	// of this rate limiter in case of collocation with other frequently accessed memory.
	padding [56]byte // cache line size - state pointer size = 64 - 8; created to avoid false sharing.

	perRequest time.Duration
	maxBurst   time.Duration
}

func (l *atomicLimiter) SpinTake(maxSpin time.Duration) bool {
	out := make(chan bool)
	go func() {
		out <- l.spinTakePointer()
	}()

	select {
	case <-time.After(maxSpin):
		return false
	case t := <-out:
		return t
	}
}

// spinTakePointer take with spin
func (l *atomicLimiter) spinTakePointer() bool {
	var (
		newState state
		taken    bool
		interval time.Duration
	)

	maxTry := 5

	for !taken && maxTry > 0 {
		maxTry--
		now := time.Now()
		previousStatePointer := atomic.LoadPointer(&l.state)
		oldState := (*state)(previousStatePointer)

		newState = state{
			last:     now,
			sleepFor: oldState.sleepFor,
		}

		// If this is our first request, then we allow it.
		if oldState.last.IsZero() {
			taken = atomic.CompareAndSwapPointer(&l.state, previousStatePointer, unsafe.Pointer(&newState))
			continue
		}

		// sleepFor calculates how much time we should sleep based on
		// the perRequest budget and how long the last request took.
		// Since the request may take longer than the budget, this number
		// can get negative, and is summed across requests.
		newState.sleepFor += l.perRequest - now.Sub(oldState.last)
		// We shouldn't allow sleepFor to get too negative, since it would mean that
		// a service that slowed down a lot for a short period of time would get
		// a much higher RPS following that.
		if newState.sleepFor < l.maxBurst {
			newState.sleepFor = l.maxBurst
		}
		if newState.sleepFor > 0 {
			newState.last = newState.last.Add(newState.sleepFor)
			interval, newState.sleepFor = newState.sleepFor, 0
		}
		taken = atomic.CompareAndSwapPointer(&l.state, previousStatePointer, unsafe.Pointer(&newState))
	}

	if taken {
		time.Sleep(interval)
	}

	return taken
}

// Take check if request can bee pass return true
func (l *atomicLimiter) Take() bool {
	var newState state

	previousStatePointer := atomic.LoadPointer(&l.state)
	now := time.Now()

	oldState := (*state)(previousStatePointer)

	newState = state{
		last:     now,
		sleepFor: oldState.sleepFor,
	}

	// If this is our first request, then we allow it.
	if oldState.last.IsZero() {
		return atomic.CompareAndSwapPointer(&l.state, previousStatePointer, unsafe.Pointer(&newState))
	}

	newState.sleepFor += l.perRequest - now.Sub(oldState.last)

	if newState.sleepFor < l.maxBurst {
		newState.sleepFor = l.maxBurst
	}

	if newState.sleepFor > 0 {
		return false
	}

	return atomic.CompareAndSwapPointer(&l.state, previousStatePointer, unsafe.Pointer(&newState))
}
