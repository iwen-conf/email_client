// Package client 提供Email gRPC服务客户端
package client

import (
	"time"

	"github.com/iwen-conf/email_client/client/conn"
	"github.com/iwen-conf/email_client/client/core"
	"github.com/iwen-conf/email_client/client/middleware"
)

// 重新导出常用类型，方便使用
type (
	// EmailClient 是客户端主要入口
	EmailClient = core.EmailClient

	// Option 定义客户端配置选项的函数类型
	Option = core.Option

	// ConnectionConfig 定义连接配置参数
	ConnectionConfig = core.ConnectionConfig

	// RetryConfig 定义重试配置参数
	RetryConfig = core.RetryConfig

	// CircuitBreakerConfig 定义断路器配置参数
	CircuitBreakerConfig = core.CircuitBreakerConfig

	// RateLimiterConfig 定义速率限制配置参数
	RateLimiterConfig = middleware.RateLimiterConfig

	// RateLimitExceededError 表示速率限制异常
	RateLimitExceededError = middleware.RateLimitExceededError

	// TLSConfig 定义TLS配置参数
	TLSConfig = conn.TLSConfig
)

// 导出常用变量和函数
var (
	// DefaultConnectionConfig 提供默认连接配置
	DefaultConnectionConfig = core.DefaultConnectionConfig

	// DefaultRetryConfig 提供默认重试配置
	DefaultRetryConfig = core.DefaultRetryConfig

	// DefaultCircuitBreakerConfig 提供默认断路器配置
	DefaultCircuitBreakerConfig = core.DefaultCircuitBreakerConfig

	// DefaultRateLimiterConfig 提供默认速率限制配置
	DefaultRateLimiterConfig = middleware.DefaultRateLimiterConfig

	// DefaultTLSConfig 提供默认TLS配置
	DefaultTLSConfig = conn.DefaultTLSConfig

	// ExponentialBackoff 实现指数退避重试策略
	ExponentialBackoff = middleware.ExponentialBackoff

	// WithConnectionConfig 设置连接相关配置
	WithConnectionConfig = core.WithConnectionConfig

	// WithRetryConfig 设置重试相关配置
	WithRetryConfig = core.WithRetryConfig

	// WithCircuitBreakerConfig 设置断路器相关配置
	WithCircuitBreakerConfig = core.WithCircuitBreakerConfig

	// WithRateLimiterConfig 设置速率限制相关配置
	WithRateLimiterConfig = core.WithRateLimiterConfig

	// WithTLSConfig 设置TLS相关配置
	WithTLSConfig = core.WithTLSConfig

	// DisableCircuitBreaker 禁用断路器
	DisableCircuitBreaker = core.DisableCircuitBreaker

	// DisableRateLimiter 禁用速率限制
	DisableRateLimiter = core.DisableRateLimiter

	// DisableTLS 禁用TLS
	DisableTLS = core.DisableTLS

	// EnableHealthCheck 启用健康检查
	EnableHealthCheck = core.EnableHealthCheck

	// DisableHealthCheck 禁用健康检查
	DisableHealthCheck = core.DisableHealthCheck
)

// NewEmailClient 创建一个新的 EmailClient 实例。
// 这是一个便捷函数，内部调用 core.NewEmailClient
func NewEmailClient(grpcAddress string, requestTimeout time.Duration, defaultPageSize int32, debug bool, opts ...Option) (*EmailClient, error) {
	return core.NewEmailClient(grpcAddress, requestTimeout, defaultPageSize, debug, opts...)
}

// NewClientMetrics 创建一个新的指标收集器
func NewClientMetrics(maxLastErrors int) *middleware.ClientMetrics {
	return middleware.NewClientMetrics(maxLastErrors)
}

// NewCircuitBreaker 创建一个新的断路器
func NewCircuitBreaker(config CircuitBreakerConfig, debug bool) *middleware.CircuitBreaker {
	return middleware.NewCircuitBreaker(middleware.CircuitBreakerConfig{
		FailureThreshold:    config.FailureThreshold,
		ResetTimeout:        config.ResetTimeout,
		HalfOpenMaxRequests: config.HalfOpenMaxRequests,
	}, debug)
}

// NewRateLimiter 创建一个新的速率限制器
func NewRateLimiter(config RateLimiterConfig, debug bool) *middleware.RateLimiter {
	return middleware.NewRateLimiter(middleware.RateLimiterConfig{
		RequestsPerSecond: config.RequestsPerSecond,
		MaxBurst:          config.MaxBurst,
		WaitTimeout:       config.WaitTimeout,
	}, debug)
}

// ErrEmptyGrpcAddress 表示提供的gRPC地址为空
var ErrEmptyGrpcAddress = core.ErrEmptyGrpcAddress
