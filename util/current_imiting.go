package util

import (
	"sync"
	"time"
)

type IPQPSLimiter struct {
	limiterMap sync.Map // 存储每个 IP 地址的 QPS 限制器
}

func NewIPQPSLimiter() *IPQPSLimiter {
	return &IPQPSLimiter{}
}

type ClientQPSLimiter struct {
	counter     int           // 当前请求数
	maxQPS      int           // 最大QPS限制
	windowSize  time.Duration // 窗口时间大小
	lock        sync.Mutex    // 互斥锁，用于保护计数器的并发访问
	resetTicker *time.Ticker  // 定时器，用于重置计数器
}

func NewClientQPSLimiter(maxQPS int, windowSize time.Duration) *ClientQPSLimiter {
	return &ClientQPSLimiter{
		maxQPS:      maxQPS,
		windowSize:  windowSize,
		resetTicker: time.NewTicker(windowSize),
	}
}
func (limiter *ClientQPSLimiter) Allow() bool {
	limiter.lock.Lock()
	defer limiter.lock.Unlock()

	limiter.counter++
	if limiter.counter > limiter.maxQPS {
		return false // 超过QPS限制，拒绝请求
	}

	return true
}

func (limiter *ClientQPSLimiter) Start() {
	go func() {
		for range limiter.resetTicker.C {
			limiter.lock.Lock()
			limiter.counter = 0 // 重置计数器
			limiter.lock.Unlock()
		}
	}()
}
func (limiter *IPQPSLimiter) Allow(ip string, maxQPS int, windowSize time.Duration) bool {
	limiterObj, _ := limiter.limiterMap.LoadOrStore(ip, NewClientQPSLimiter(maxQPS, windowSize))
	clientLimiter := limiterObj.(*ClientQPSLimiter)
	clientLimiter.Start()
	return clientLimiter.Allow()
}
