package middleware

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimitExceededError 表示超出了速率限制
type RateLimitExceededError struct {
	RequestsPerSecond float64
	Message           string
}

// Error 实现 error 接口
func (e *RateLimitExceededError) Error() string {
	return fmt.Sprintf("速率限制超出 (%.2f requests/s): %s", e.RequestsPerSecond, e.Message)
}

// RateLimiter 使用令牌桶算法实现速率限制
type RateLimiter struct {
	mu             sync.Mutex
	tokens         float64       // 当前可用的令牌数
	maxTokens      float64       // 令牌桶容量
	refillRate     float64       // 每秒填充的令牌数（请求/秒）
	lastRefillTime time.Time     // 上次填充时间
	waitTimeout    time.Duration // 等待令牌的超时时间
	debug          bool
}

// RateLimiterConfig 定义速率限制器配置
type RateLimiterConfig struct {
	// 每秒允许的请求数
	RequestsPerSecond float64
	// 同一时刻允许的最大并发请求数
	MaxBurst float64
	// 等待令牌的超时时间，如果为0则不等待
	WaitTimeout time.Duration
}

// DefaultRateLimiterConfig 返回默认的速率限制器配置
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		RequestsPerSecond: 10.0,
		MaxBurst:          20.0,
		WaitTimeout:       0,
	}
}

// NewRateLimiter 创建一个新的速率限制器
func NewRateLimiter(config RateLimiterConfig, debug bool) *RateLimiter {
	// 检查配置的有效性
	if config.RequestsPerSecond <= 0 {
		config.RequestsPerSecond = 1.0
	}
	if config.MaxBurst <= 0 {
		config.MaxBurst = config.RequestsPerSecond
	}

	return &RateLimiter{
		tokens:         config.MaxBurst,
		maxTokens:      config.MaxBurst,
		refillRate:     config.RequestsPerSecond,
		lastRefillTime: time.Now(),
		waitTimeout:    config.WaitTimeout,
		debug:          debug,
	}
}

// Allow 检查是否允许新请求，不阻塞
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refillTokens()

	if rl.tokens >= 1 {
		rl.tokens--
		return true
	}

	return false
}

// Wait 等待直到有令牌可用或超时
func (rl *RateLimiter) Wait(ctx context.Context) error {
	if rl.waitTimeout <= 0 {
		// 不等待，直接尝试获取令牌
		if !rl.Allow() {
			return &RateLimitExceededError{
				RequestsPerSecond: rl.refillRate,
				Message:           "已达到速率限制且不等待",
			}
		}
		return nil
	}

	// 创建带超时的上下文
	waitCtx := ctx
	if deadline, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithTimeout(ctx, rl.waitTimeout)
		defer cancel()
	} else {
		// 使用较短的超时时间
		remaining := time.Until(deadline)
		if remaining > rl.waitTimeout {
			remaining = rl.waitTimeout
		}
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithTimeout(ctx, remaining)
		defer cancel()
	}

	// 尝试获取令牌，有超时限制
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		if rl.Allow() {
			return nil
		}

		select {
		case <-waitCtx.Done():
			return &RateLimitExceededError{
				RequestsPerSecond: rl.refillRate,
				Message:           "等待可用令牌超时",
			}
		case <-ticker.C:
			// 继续尝试
		}
	}
}

// refillTokens 根据上次填充后经过的时间填充令牌
func (rl *RateLimiter) refillTokens() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefillTime).Seconds()

	if elapsed > 0 {
		// 计算需要添加的令牌
		newTokens := elapsed * rl.refillRate

		// 更新令牌数，不超过最大值
		rl.tokens += newTokens
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}

		// 更新上次填充时间
		rl.lastRefillTime = now
	}
}

// GetCurrentRate 获取当前速率限制值（令牌/秒）
func (rl *RateLimiter) GetCurrentRate() float64 {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.refillRate
}

// GetAvailableTokens 获取当前可用令牌数
func (rl *RateLimiter) GetAvailableTokens() float64 {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.refillTokens()
	return rl.tokens
}

// SetRate 动态调整速率限制
func (rl *RateLimiter) SetRate(requestsPerSecond float64) {
	if requestsPerSecond <= 0 {
		return
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// 先更新现有令牌
	rl.refillTokens()

	// 更新速率
	rl.refillRate = requestsPerSecond

	// 可选：调整最大令牌数
	if requestsPerSecond > rl.maxTokens {
		rl.maxTokens = requestsPerSecond
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}
	}
}
