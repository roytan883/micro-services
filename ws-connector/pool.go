package main

import (
	"container/list"
	"sync"
	"sync/atomic"
	"time"
)

//PoolHander ...
type PoolHander func(data interface{})

//RunGoPool ...
type RunGoPool struct {
	name            string
	rate            *RateLimiter
	dataChan        chan interface{}
	dataChanLen     int64
	waitDataChanLen int64
	hander          PoolHander
	stopChan        chan int
	stoped          uint32
}

// NewRunGoPool creates a new rate limiter for the limit and interval.
func NewRunGoPool(name string, limit int, interval time.Duration, hander PoolHander) *RunGoPool {
	pool := &RunGoPool{
		name:     name,
		hander:   hander,
		stopChan: make(chan int, 1),
		dataChan: make(chan interface{}, 10000), //若缓存太大会开销太多内存
	}
	pool.rate = NewRateLimiter(limit, interval)
	return pool
}

//Start ...
func (p *RunGoPool) Start() {
	atomic.StoreUint32(&p.stoped, 0)
	go func() {
		for {
			select {
			case <-p.stopChan:
				return
			case data := <-p.dataChan:
				atomic.AddInt64(&p.dataChanLen, -1)
				p.rate.Wait()
				go p.hander(data)
			}
		}
	}()
}

//Stop ...
func (p *RunGoPool) Stop() {
	atomic.AddUint32(&p.stoped, 1)
	p.stopChan <- 1
}

//Add ...
func (p *RunGoPool) Add(data interface{}) {
	if atomic.LoadUint32(&p.stoped) > 0 {
		return
	}
	//dataChan最大缓存10000，所以9000以下的时候，直接放进去
	//省掉了开go协程的开销
	if atomic.LoadInt64(&p.dataChanLen) < 9000 {
		p.dataChan <- data
		atomic.AddInt64(&p.dataChanLen, 1)
	} else {
		if atomic.LoadInt64(&p.waitDataChanLen) > 20000 {
			log.Warnf("RunGoPool[%s] Add: waitDataChanLen > 20000, reject\n", p.name)
			return
		}
		//dataChan最大缓存10000，所以9000以上的时候
		//可能dataChan缓存已经满了，所以要开一个go 协程写入，当满缓存的时候不会卡住外面
		log.Warnf("RunGoPool[%s] Add: dataChanLen = %d\n", p.name, atomic.LoadInt64(&p.dataChanLen))
		atomic.AddInt64(&p.waitDataChanLen, 1)
		log.Warnf("RunGoPool[%s] Add: waitDataChanLen = %d\n", p.name, atomic.LoadInt64(&p.waitDataChanLen))
		go func() {
			p.dataChan <- data
			atomic.AddInt64(&p.dataChanLen, 1)
			atomic.AddInt64(&p.waitDataChanLen, -1)
		}()
	}
}

//RunTimerPool ...
type RunTimerPool struct {
	rate     *RateLimiter
	queue    *list.List
	mtx      sync.RWMutex
	hander   PoolHander
	stopChan chan int
}

// NewRunTimerPool creates a new rate limiter for the limit and interval(>100ms).
func NewRunTimerPool(limit int, interval time.Duration, hander PoolHander) *RunTimerPool {
	pool := &RunTimerPool{
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
func (p *RunTimerPool) Start() {
	go func() {
		ticker := time.NewTicker(time.Millisecond * 1)
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
							log.Info("RunTimerPool limited, remain len = ", p.queue.Len())
							p.mtx.RUnlock()
						}
					}
				}
			}
		}
	}()
}

//Stop ...
func (p *RunTimerPool) Stop() {
	p.stopChan <- 1
}

//Add ...
func (p *RunTimerPool) Add(data interface{}) {
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
