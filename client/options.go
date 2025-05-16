package client

import (
	"time"
)

// Option 定义客户端配置选项的函数类型
type Option func(*clientOptions)

// clientOptions 包含所有可配置的客户端选项
type clientOptions struct {
	// 连接相关选项
	initialBackoff      time.Duration // 初始重试延迟
	maxBackoff          time.Duration // 最大重试延迟
	backoffMultiplier   float64       // 重试延迟增长因子
	minConnectTimeout   time.Duration // 最小连接超时
	enableHealthCheck   bool          // 是否启用健康检查
	healthCheckInterval time.Duration // 健康检查间隔

	// 重试相关选项
	maxRetries  int           // 最大重试次数
	retryDelay  time.Duration // 重试延迟
	retryPolicy RetryPolicy   // 重试策略

	// 断路器相关选项
	enableCircuitBreaker bool          // 是否启用断路器
	failureThreshold     int           // 故障阈值
	circuitResetTimeout  time.Duration // 断路器重置超时
	halfOpenMaxRequests  int           // 半开状态最大请求数
}

// RetryPolicy 定义重试策略函数类型
type RetryPolicy func(attempt int) time.Duration

// 默认选项
var defaultOptions = clientOptions{
	initialBackoff:      1 * time.Second,
	maxBackoff:          30 * time.Second,
	backoffMultiplier:   1.5,
	minConnectTimeout:   20 * time.Second,
	enableHealthCheck:   true,
	healthCheckInterval: 30 * time.Second,

	maxRetries:  3,
	retryDelay:  500 * time.Millisecond,
	retryPolicy: ExponentialBackoff,

	enableCircuitBreaker: false,
	failureThreshold:     5,
	circuitResetTimeout:  10 * time.Second,
	halfOpenMaxRequests:  1,
}

// WithConnectionConfig 设置连接相关配置
func WithConnectionConfig(config ConnectionConfig) Option {
	return func(opts *clientOptions) {
		opts.initialBackoff = config.InitialBackoff
		opts.maxBackoff = config.MaxBackoff
		opts.backoffMultiplier = config.BackoffMultiplier
		opts.minConnectTimeout = config.MinConnectTimeout
		opts.enableHealthCheck = config.EnableHealthCheck
		opts.healthCheckInterval = config.HealthCheckInterval
	}
}

// ConnectionConfig 定义连接配置参数
type ConnectionConfig struct {
	InitialBackoff      time.Duration // 初始重试延迟
	MaxBackoff          time.Duration // 最大重试延迟
	BackoffMultiplier   float64       // 重试延迟增长因子
	MinConnectTimeout   time.Duration // 最小连接超时
	EnableHealthCheck   bool          // 是否启用健康检查
	HealthCheckInterval time.Duration // 健康检查间隔
}

// DefaultConnectionConfig 提供默认连接配置
var DefaultConnectionConfig = ConnectionConfig{
	InitialBackoff:      1 * time.Second,
	MaxBackoff:          30 * time.Second,
	BackoffMultiplier:   1.5,
	MinConnectTimeout:   20 * time.Second,
	EnableHealthCheck:   true,
	HealthCheckInterval: 30 * time.Second,
}

// WithRetryConfig 设置重试相关配置
func WithRetryConfig(config RetryConfig) Option {
	return func(opts *clientOptions) {
		opts.maxRetries = config.MaxRetries
		opts.retryDelay = config.RetryDelay
		if config.RetryPolicy != nil {
			opts.retryPolicy = config.RetryPolicy
		}
	}
}

// RetryConfig 定义重试配置参数
type RetryConfig struct {
	MaxRetries  int           // 最大重试次数
	RetryDelay  time.Duration // 重试延迟
	RetryPolicy RetryPolicy   // 重试策略函数
}

// DefaultRetryConfig 提供默认重试配置
var DefaultRetryConfig = RetryConfig{
	MaxRetries:  3,
	RetryDelay:  500 * time.Millisecond,
	RetryPolicy: ExponentialBackoff,
}

// ExponentialBackoff 实现指数退避重试策略
func ExponentialBackoff(attempt int) time.Duration {
	return time.Duration(1<<uint(attempt)) * 100 * time.Millisecond
}

// WithCircuitBreakerConfig 设置断路器相关配置
func WithCircuitBreakerConfig(config CircuitBreakerConfig) Option {
	return func(opts *clientOptions) {
		opts.enableCircuitBreaker = true
		opts.failureThreshold = config.FailureThreshold
		opts.circuitResetTimeout = config.ResetTimeout
		opts.halfOpenMaxRequests = config.HalfOpenMaxRequests
	}
}

// DisableCircuitBreaker 禁用断路器
func DisableCircuitBreaker() Option {
	return func(opts *clientOptions) {
		opts.enableCircuitBreaker = false
	}
}

// CircuitBreakerConfig 定义断路器配置参数
type CircuitBreakerConfig struct {
	FailureThreshold    int           // 连续失败次数阈值
	ResetTimeout        time.Duration // 断路器从开到半开的重置时间
	HalfOpenMaxRequests int           // 半开状态下允许的最大请求数
}

// DefaultCircuitBreakerConfig 提供默认断路器配置
var DefaultCircuitBreakerConfig = CircuitBreakerConfig{
	FailureThreshold:    5,
	ResetTimeout:        10 * time.Second,
	HalfOpenMaxRequests: 1,
}

// EnableHealthCheck 启用健康检查
func EnableHealthCheck(interval time.Duration) Option {
	return func(opts *clientOptions) {
		opts.enableHealthCheck = true
		if interval > 0 {
			opts.healthCheckInterval = interval
		}
	}
}

// DisableHealthCheck 禁用健康检查
func DisableHealthCheck() Option {
	return func(opts *clientOptions) {
		opts.enableHealthCheck = false
	}
}
