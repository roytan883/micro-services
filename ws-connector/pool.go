package main

import (
	"container/list"
	"sync"
	"time"
)

//PoolHander ...
type PoolHander func(data interface{})

//RunPool ...
type RunPool struct {
	rate     *RateLimiter
	queue    *list.List
	mtx      sync.RWMutex
	hander   PoolHander
	stopChan chan int
}

// NewRunPool creates a new rate limiter for the limit and interval(>100ms).
func NewRunPool(limit int, interval time.Duration, hander PoolHander) *RunPool {
	pool := &RunPool{
		hander:   hander,
		stopChan: make(chan int, 1),
	}
	if interval < time.Millisecond*100 {
		interval = time.Millisecond * 100
	}
	pool.rate = NewRateLimiter(limit, interval)
	pool.queue = list.New()
	return pool
}

//Start ...
func (p *RunPool) Start() {
	go func() {
		ticker := time.NewTicker(time.Millisecond * 50)
		for {
			select {
			case <-p.stopChan:
				ticker.Stop()
				return
			case <-ticker.C:
				p.mtx.RLock()
				len := p.queue.Len()
				p.mtx.RUnlock()
				if len > 0 {
					ok := true
					for ok {
						ok, _ = p.rate.Try()
						if ok {
							p.mtx.RLock()
							item := p.queue.Front()
							p.mtx.RUnlock()
							if item != nil {
								data := item.Value
								p.mtx.Lock()
								p.queue.Remove(item)
								p.mtx.Unlock()
								go p.hander(data)
							} else {
								ok = false
								break
							}
						} else {
							p.mtx.RLock()
							log.Info("RunPool limited, remain len = ", p.queue.Len())
							p.mtx.RUnlock()
						}
					}
				}
			}
		}
	}()
}

//Stop ...
func (p *RunPool) Stop() {
	p.stopChan <- 1
}

//Add ...
func (p *RunPool) Add(data interface{}) {
	p.mtx.Lock()
	p.queue.PushBack(data)
	p.mtx.Unlock()
}

// A RateLimiter limits the rate at which an action can be performed.  It
// applies neither smoothing (like one could achieve in a token bucket system)
// nor does it offer any conception of warmup, wherein the rate of actions
// granted are steadily increased until a steady throughput equilibrium is
// reached.
type RateLimiter struct {
	limit    int
	interval time.Duration
	mtx      sync.Mutex
	times    list.List
}

// NewRateLimiter creates a new rate limiter for the limit and interval.
func NewRateLimiter(limit int, interval time.Duration) *RateLimiter {
	lim := &RateLimiter{
		limit:    limit,
		interval: interval,
	}
	lim.times.Init()
	return lim
}

// Wait blocks if the rate limit has been reached.  Wait offers no guarantees
// of fairness for multiple actors if the allowed rate has been temporarily
// exhausted.
func (r *RateLimiter) Wait() {
	for {
		ok, remaining := r.Try()
		if ok {
			break
		}
		time.Sleep(remaining)
	}
}

// Try returns true if under the rate limit, or false if over and the
// remaining time before the rate limit expires.
func (r *RateLimiter) Try() (ok bool, remaining time.Duration) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	now := time.Now()
	if l := r.times.Len(); l < r.limit {
		r.times.PushBack(now)
		return true, 0
	}
	frnt := r.times.Front()
	if diff := now.Sub(frnt.Value.(time.Time)); diff < r.interval {
		return false, r.interval - diff
	}
	frnt.Value = now
	r.times.MoveToBack(frnt)
	return true, 0
}
