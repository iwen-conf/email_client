package main

import (
	"strings"
	"testing"
	"time"

	"github.com/iwen-conf/email_client/client"
	"github.com/iwen-conf/email_client/client/logger"
)

func TestClientIntegration(t *testing.T) {
	// 如果是真实的集成测试，应该使用实际的gRPC服务地址
	// 这里使用一个不存在的地址，期望会连接失败，但我们可以测试配置是否正确设置
	grpcAddress := "localhost:50051"
	requestTimeout := 5 * time.Second
	defaultPageSize := int32(20)
	debug := true

	t.Run("基本客户端创建", func(t *testing.T) {
		// 这个测试会失败，因为没有实际的服务运行
		// 但我们可以检查错误消息是否符合预期
		_, err := client.NewEmailClient(grpcAddress, requestTimeout, defaultPageSize, debug)
		if err == nil {
			t.Errorf("期望连接错误，但没有得到错误")
		}
		// 这里可以检查错误类型或消息
	})

	t.Run("速率限制器", func(t *testing.T) {
		// 测试速率限制器
		config := client.DefaultRateLimiterConfig()
		config.RequestsPerSecond = 10
		config.MaxBurst = 5

		limiter := client.NewRateLimiter(config, true)

		// 测试允许请求
		for i := 0; i < 5; i++ {
			if !limiter.Allow() {
				t.Errorf("前5个请求应该被允许")
			}
		}

		// 当前桶应该为空，下一个请求应该被拒绝
		if limiter.Allow() {
			t.Errorf("第6个请求应该被拒绝")
		}

		// 等待一段时间，应该有新的令牌生成
		time.Sleep(time.Second) // 等待1秒，应该生成10个新令牌

		// 测试桶是否已经重新填充
		if !limiter.Allow() {
			t.Errorf("等待后应该可以获取新令牌")
		}
	})

	t.Run("断路器", func(t *testing.T) {
		config := client.CircuitBreakerConfig{
			FailureThreshold:    3,
			ResetTimeout:        2 * time.Second,
			HalfOpenMaxRequests: 1,
		}

		cb := client.NewCircuitBreaker(config, true)

		// 断路器应该初始为闭合状态
		if !cb.IsAllowed() {
			t.Errorf("初始状态应该允许请求")
		}

		// 记录3次失败，断路器应该打开
		for i := 0; i < 3; i++ {
			cb.OnFailure()
		}

		// 断路器应该处于打开状态，拒绝请求
		if cb.IsAllowed() {
			t.Errorf("多次失败后应该拒绝请求")
		}

		// 等待重置时间过后，断路器应该进入半开状态
		time.Sleep(2 * time.Second)

		// 断路器处于半开状态，应该允许一个请求
		if !cb.IsAllowed() {
			t.Errorf("重置后应该允许一个请求")
		}

		// 第二个请求应该被拒绝
		if cb.IsAllowed() {
			t.Errorf("半开状态下第二个请求应该被拒绝")
		}

		// 记录一次成功，断路器应该关闭
		cb.OnSuccess()

		// 断路器应该回到闭合状态，允许请求
		if !cb.IsAllowed() {
			t.Errorf("成功后应该允许请求")
		}
	})

	t.Run("结构化日志", func(t *testing.T) {
		// 创建一个自定义的buffer来捕获日志输出
		buffer := &testLogBuffer{}

		// 创建日志实例
		log := logger.NewStandardLogger()
		log.SetOutput(buffer)
		log.SetLevel(logger.InfoLevel)

		// 记录一条测试日志
		log.Info("测试日志")

		// 验证日志输出
		if !buffer.Contains("测试日志") {
			t.Errorf("日志应该包含'测试日志'")
		}

		// 测试级别过滤
		buffer.Reset()
		log.Debug("调试信息")

		// Debug级别应该被过滤掉
		if buffer.Contains("调试信息") {
			t.Errorf("调试日志不应该被输出")
		}

		// 设置为Debug级别
		log.SetLevel(logger.DebugLevel)
		log.Debug("现在可以看到调试信息")

		// 此时应该可以看到Debug级别的日志
		if !buffer.Contains("现在可以看到调试信息") {
			t.Errorf("调试日志应该被输出")
		}
	})
}

// 简单的日志缓冲区用于测试
type testLogBuffer struct {
	content string
}

func (b *testLogBuffer) Write(p []byte) (n int, err error) {
	b.content += string(p)
	return len(p), nil
}

func (b *testLogBuffer) Contains(s string) bool {
	return strings.Contains(b.content, s)
}

func (b *testLogBuffer) Reset() {
	b.content = ""
}
