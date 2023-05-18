package util

import (
	"fmt"
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
	counter      int           // 当前请求数
	maxQPS       int           // 最大QPS限制
	windowSize   time.Duration // 窗口时间大小
	lock         sync.Mutex    // 互斥锁，用于保护计数器的并发访问
	resetTicker  *time.Ticker  // 定时器，用于重置计数器
	deleteTicker *time.Ticker  // 定时查看当前请求数是否为0
}

func NewClientQPSLimiter(maxQPS int, windowSize, deleteSize time.Duration) *ClientQPSLimiter {
	return &ClientQPSLimiter{
		maxQPS:       maxQPS,
		windowSize:   windowSize,
		resetTicker:  time.NewTicker(windowSize),
		deleteTicker: time.NewTicker(deleteSize),
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
func (limiter *IPQPSLimiter) Allow(ip string, maxQPS int, windowSize, deleteSize time.Duration) bool {
	limiterObj, _ := limiter.limiterMap.LoadOrStore(ip, NewClientQPSLimiter(maxQPS, windowSize, deleteSize))
	clientLimiter := limiterObj.(*ClientQPSLimiter)
	clientLimiter.Start()
	limiter.DeleteMap(ip, clientLimiter)
	return clientLimiter.Allow()
}
func (limiter *IPQPSLimiter) DeleteMap(ip string, clientLimiter *ClientQPSLimiter) {
	go func() {
		for range clientLimiter.deleteTicker.C {
			if clientLimiter.counter == 0 {
				limiter.limiterMap.Delete(ip)
				fmt.Printf("delete key is :%s\n", ip)
				clientLimiter.deleteTicker.Stop()
			}
		}
	}()
}
