package client

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// 定义可重试的错误类型
var retryableCodes = map[codes.Code]bool{
	codes.Unavailable:       true, // 服务不可用
	codes.DeadlineExceeded:  true, // 请求超时
	codes.ResourceExhausted: true, // 资源耗尽
	codes.Aborted:           true, // 操作被终止
	codes.Internal:          true, // 内部服务器错误
}

// WithRetryAndMetrics 创建一个带有重试和指标收集的一元 gRPC 拦截器
func WithRetryAndMetrics(debug bool, metrics *ClientMetrics, retryConfig RetryConfig) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, resp interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		var lastErr error
		startTime := time.Now()

		// 如果未指定最大重试次数，则默认为不重试
		maxRetries := retryConfig.MaxRetries
		if maxRetries <= 0 {
			// 单次调用，不重试
			err := invoker(ctx, method, req, resp, cc, opts...)
			if metrics != nil {
				metrics.RecordRequest(err == nil, time.Since(startTime))
			}
			return err
		}

		// 执行重试逻辑
		for attempt := 0; attempt <= maxRetries; attempt++ {
			// 如果不是第一次尝试，则根据重试策略进行等待
			if attempt > 0 {
				delay := retryConfig.RetryDelay
				if retryConfig.RetryPolicy != nil {
					delay = retryConfig.RetryPolicy(attempt)
				}

				if debug {
					log.Printf("[DEBUG] Retry: 第 %d 次重试 method=%s, 延迟=%v", attempt, method, delay)
				}

				select {
				case <-time.After(delay):
					// 延迟后继续
				case <-ctx.Done():
					// 如果上下文被取消，则返回上下文错误
					if metrics != nil {
						metrics.RecordRequest(false, time.Since(startTime))
					}
					return ctx.Err()
				}
			}

			// 调用实际的 gRPC 方法
			err := invoker(ctx, method, req, resp, cc, opts...)
			if err == nil {
				// 成功，记录指标并返回
				if metrics != nil {
					metrics.RecordRequest(true, time.Since(startTime))
				}
				return nil
			}

			// 判断是否是可重试的错误
			if !isRetryableError(err) {
				// 不可重试的错误，记录指标并立即返回
				if metrics != nil {
					metrics.RecordRequest(false, time.Since(startTime))
				}
				return err
			}

			// 记录最后一次遇到的错误，如果所有重试都失败，将返回此错误
			lastErr = err

			// 最后一次重试也失败了
			if attempt == maxRetries {
				if debug {
					log.Printf("[DEBUG] Retry: 已达到最大重试次数 %d, method=%s", maxRetries, method)
				}
				break
			}
		}

		// 记录失败指标
		if metrics != nil {
			metrics.RecordRequest(false, time.Since(startTime))
		}

		// 所有重试都失败，返回包装后的错误
		return fmt.Errorf("在 %d 次尝试后调用失败: %w", retryConfig.MaxRetries+1, lastErr)
	}
}

// isRetryableError 判断错误是否可重试
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// 检查是否是 gRPC 错误
	if st, ok := status.FromError(err); ok {
		return retryableCodes[st.Code()]
	}

	// 检查是否是连接错误
	return errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled) ||
		isConnectionError(err)
}

// isConnectionError 判断是否是连接相关错误
func isConnectionError(err error) bool {
	// 检查错误字符串中是否包含连接相关的关键词
	// 注意：这种方式不是最准确的，但在没有统一错误类型的情况下是常用的方法
	if err == nil {
		return false
	}

	errMsg := err.Error()
	connectionErrKeywords := []string{
		"connection",
		"connectivity",
		"transport",
		"broken pipe",
		"reset by peer",
		"timeout",
		"deadline",
		"closed",
	}

	for _, keyword := range connectionErrKeywords {
		// 检查错误消息中是否包含关键词
		if strings.Contains(errMsg, keyword) {
			return true
		}
		// 检查是否是常见的上下文错误
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return true
		}
	}

	return false
}

// RetryableFn 定义可重试的函数类型
type RetryableFn func() error

// RetryWithBackoff 使用退避策略重试函数
func RetryWithBackoff(fn RetryableFn, config RetryConfig, debug bool) error {
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := config.RetryDelay
			if config.RetryPolicy != nil {
				delay = config.RetryPolicy(attempt)
			}

			if debug {
				log.Printf("[DEBUG] RetryWithBackoff: 第 %d 次重试, 延迟=%v", attempt, delay)
			}

			time.Sleep(delay)
		}

		err := fn()
		if err == nil {
			return nil // 成功
		}

		// 判断错误是否可重试
		if !isRetryableError(err) {
			return err // 不可重试的错误，立即返回
		}

		lastErr = err

		// 最后一次重试也失败了
		if attempt == config.MaxRetries {
			if debug {
				log.Printf("[DEBUG] RetryWithBackoff: 已达到最大重试次数 %d", config.MaxRetries)
			}
			break
		}
	}

	// 所有重试都失败
	return fmt.Errorf("在 %d 次尝试后操作失败: %w", config.MaxRetries+1, lastErr)
}
