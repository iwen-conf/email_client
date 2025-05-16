package client

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// CircuitState 表示断路器的状态
type CircuitState int

const (
	CircuitClosed   CircuitState = iota // 关闭状态，请求正常通过
	CircuitOpen                         // 开路状态，直接拒绝请求
	CircuitHalfOpen                     // 半开状态，允许有限的请求通过以测试服务是否恢复
)

// String 返回断路器状态的字符串表示
func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "CLOSED"
	case CircuitOpen:
		return "OPEN"
	case CircuitHalfOpen:
		return "HALF_OPEN"
	default:
		return fmt.Sprintf("未知状态(%d)", int(s))
	}
}

// CircuitBreaker 实现断路器模式
type CircuitBreaker struct {
	mutex               sync.RWMutex
	state               CircuitState  // 当前状态
	failureThreshold    int           // 连续失败阈值
	resetTimeout        time.Duration // 从开到半开的重置时间
	halfOpenMaxRequests int           // 半开状态下的最大请求数
	failureCount        int           // 当前连续失败计数
	successCount        int           // 半开状态下成功计数
	lastFailureTime     time.Time     // 最后一次失败时间
	requestCount        int           // 半开状态下请求计数
	openStateListeners  []func()      // 断路器打开时的回调函数
	closeStateListeners []func()      // 断路器关闭时的回调函数
	debug               bool          // 是否启用调试日志
}

// NewCircuitBreaker 创建一个新的断路器
func NewCircuitBreaker(config CircuitBreakerConfig, debug bool) *CircuitBreaker {
	cb := &CircuitBreaker{
		state:               CircuitClosed,
		failureThreshold:    config.FailureThreshold,
		resetTimeout:        config.ResetTimeout,
		halfOpenMaxRequests: config.HalfOpenMaxRequests,
		debug:               debug,
	}

	if cb.halfOpenMaxRequests <= 0 {
		cb.halfOpenMaxRequests = 1 // 确保允许至少一个请求
	}

	return cb
}

// AddOpenStateListener 添加断路器打开时的监听器
func (cb *CircuitBreaker) AddOpenStateListener(listener func()) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.openStateListeners = append(cb.openStateListeners, listener)
}

// AddCloseStateListener 添加断路器关闭时的监听器
func (cb *CircuitBreaker) AddCloseStateListener(listener func()) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.closeStateListeners = append(cb.closeStateListeners, listener)
}

// IsAllowed 判断当前请求是否被允许通过断路器
func (cb *CircuitBreaker) IsAllowed() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()

	switch cb.state {
	case CircuitClosed:
		return true // 关闭状态，所有请求都允许通过

	case CircuitOpen:
		// 检查是否达到重置超时时间，如果是则转为半开状态
		if cb.lastFailureTime.Add(cb.resetTimeout).Before(now) {
			if cb.debug {
				log.Printf("[INFO] CircuitBreaker: 从 OPEN 状态转为 HALF_OPEN 状态")
			}
			cb.state = CircuitHalfOpen
			cb.requestCount = 0
			cb.successCount = 0
			return true // 允许第一个请求通过测试
		}
		return false // 开路状态，拒绝请求

	case CircuitHalfOpen:
		// 半开状态下控制通过的请求数量
		if cb.requestCount < cb.halfOpenMaxRequests {
			cb.requestCount++
			return true
		}
		return false

	default:
		return false
	}
}

// OnSuccess 记录请求成功
func (cb *CircuitBreaker) OnSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	switch cb.state {
	case CircuitClosed:
		// 重置失败计数
		cb.failureCount = 0

	case CircuitHalfOpen:
		// 增加成功计数
		cb.successCount++

		// 如果半开状态下成功数达到阈值，则关闭断路器
		if cb.successCount >= cb.halfOpenMaxRequests {
			if cb.debug {
				log.Printf("[INFO] CircuitBreaker: 从 HALF_OPEN 状态转为 CLOSED 状态")
			}
			cb.state = CircuitClosed
			cb.failureCount = 0

			// 通知监听器
			for _, listener := range cb.closeStateListeners {
				go listener() // 使用 goroutine 避免阻塞
			}
		}
	}
}

// OnFailure 记录请求失败
func (cb *CircuitBreaker) OnFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.lastFailureTime = time.Now()

	switch cb.state {
	case CircuitClosed:
		// 增加失败计数
		cb.failureCount++

		// 如果失败次数达到阈值，则打开断路器
		if cb.failureCount >= cb.failureThreshold {
			if cb.debug {
				log.Printf("[INFO] CircuitBreaker: 从 CLOSED 状态转为 OPEN 状态，失败次数: %d", cb.failureCount)
			}
			cb.state = CircuitOpen

			// 通知监听器
			for _, listener := range cb.openStateListeners {
				go listener() // 使用 goroutine 避免阻塞
			}
		}

	case CircuitHalfOpen:
		// 半开状态下如果失败，立即回到打开状态
		if cb.debug {
			log.Printf("[INFO] CircuitBreaker: 从 HALF_OPEN 状态转为 OPEN 状态，请求失败")
		}
		cb.state = CircuitOpen

		// 通知监听器
		for _, listener := range cb.openStateListeners {
			go listener() // 使用 goroutine 避免阻塞
		}
	}
}

// State 返回当前断路器状态
func (cb *CircuitBreaker) State() CircuitState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// Reset 重置断路器状态
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	prevState := cb.state
	cb.state = CircuitClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.requestCount = 0

	if cb.debug {
		log.Printf("[INFO] CircuitBreaker: 断路器已重置，从 %s 状态变为 CLOSED", prevState)
	}

	// 如果之前状态是打开的，通知关闭监听器
	if prevState == CircuitOpen {
		for _, listener := range cb.closeStateListeners {
			go listener()
		}
	}
}
