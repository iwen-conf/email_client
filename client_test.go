package main

import (
	"context"
	"os"
	"path/filepath"
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

// Example_SendingEmailsWithAttachments 展示如何发送带附件的邮件
func Example_SendingEmailsWithAttachments() {
	// 创建一个邮件客户端
	grpcAddress := "email-service:50051" // 实际使用时替换为真实地址
	requestTimeout := 10 * time.Second
	defaultPageSize := int32(20)
	debug := true

	client, err := client.NewEmailClient(grpcAddress, requestTimeout, defaultPageSize, debug)
	if err != nil {
		// 在实际应用中，应适当处理错误
		return
	}
	defer client.Close()

	// 准备邮件参数
	title := "测试带附件的邮件"
	content := []byte("这是一封测试邮件，包含附件。")
	from := "sender@example.com"
	to := []string{"recipient@example.com"}
	configID := "email_config_id" // 替换为实际的邮件配置ID

	// 示例1：发送单个附件的邮件
	ctx := context.Background()
	attachmentPath := "/path/to/document.pdf" // 替换为实际的文件路径

	response, err := client.EmailService().SendEmailWithAttachment(
		ctx, title, content, from, to, configID, attachmentPath,
	)
	if err != nil {
		// 处理错误
		return
	}

	if response.Success {
		// 邮件发送成功，处理成功情况
	}

	// 示例2：发送多个附件的邮件
	attachmentPaths := []string{
		"/path/to/document1.pdf",
		"/path/to/image.jpg",
		"/path/to/spreadsheet.xlsx",
	}

	response, err = client.EmailService().SendEmailWithAttachments(
		ctx, title, content, from, to, configID, attachmentPaths,
	)
	if err != nil {
		// 处理错误
		return
	}

	if response.Success {
		// 邮件发送成功，处理成功情况
	}
}

// TestSendEmailWithAttachments 测试发送带附件的功能
func TestSendEmailWithAttachments(t *testing.T) {
	// 由于该测试需要实际的服务器连接和文件，这里我们只验证文件处理逻辑
	tempDir, err := os.MkdirTemp("", "email-attachments-test")
	if err != nil {
		t.Fatalf("无法创建临时目录: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试文件
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := []byte("测试附件内容")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("无法创建测试文件: %v", err)
	}

	// 在实际测试中，这里会连接到真实的gRPC服务
	// 这里我们跳过实际的发送，只验证了文件是否存在和可读
	_, err = os.Stat(testFile)
	if err != nil {
		t.Errorf("测试文件不存在: %v", err)
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("无法读取测试文件: %v", err)
	}

	if string(content) != string(testContent) {
		t.Errorf("文件内容不匹配: 期望 %q, 得到 %q", testContent, content)
	}
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
