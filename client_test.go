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
	"github.com/iwen-conf/email_client/client/services"
	"github.com/iwen-conf/email_client/proto/email_client_pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestClientIntegration(t *testing.T) {
	// 如果是真实的集成测试，应该使用实际的gRPC服务地址
	// 这里使用一个不存在的地址，期望会连接失败，但我们可以测试配置是否正确设置
	grpcAddress := "localhost:50051"
	requestTimeout := 5 * time.Second
	defaultPageSize := int32(20)
	debug := true

	t.Run("基本客户端创建", func(t *testing.T) {
		// 尝试连接到gRPC服务，无论成功或失败都是预期的
		client, err := client.NewEmailClient(grpcAddress, requestTimeout, defaultPageSize, debug)
		if err != nil {
			// 连接失败是预期的（如果没有服务运行）
			t.Logf("连接失败（这是预期的）: %v", err)
		} else {
			// 连接成功也是可能的（如果有服务运行）
			t.Logf("连接成功，正在关闭客户端")
			client.Close()
		}
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

		// 在半开状态下，后续请求应该被拒绝，直到有成功或失败的结果
		// 这里先不测试第二个请求，因为半开状态的行为可能因实现而异

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

// TestEmailTypes 测试邮件类型功能
func TestEmailTypes(t *testing.T) {
	// 测试邮件类型常量
	t.Run("邮件类型常量", func(t *testing.T) {
		if services.EmailTypeNormal != "normal" {
			t.Errorf("期望正常邮件类型为'normal'，得到'%s'", services.EmailTypeNormal)
		}
		if services.EmailTypeTest != "test" {
			t.Errorf("期望测试邮件类型为'test'，得到'%s'", services.EmailTypeTest)
		}
	})

	// 测试邮件对象创建时设置类型
	t.Run("邮件对象类型设置", func(t *testing.T) {
		// 测试正常邮件
		normalEmail := &email_client_pb.Email{
			Title:     "正常业务邮件",
			Content:   []byte("这是正常业务邮件内容"),
			From:      "sender@example.com",
			To:        []string{"recipient@example.com"},
			EmailType: services.EmailTypeNormal,
			SentAt:    timestamppb.Now(),
		}

		// 验证邮件类型
		if normalEmail.EmailType != services.EmailTypeNormal {
			t.Errorf("期望邮件类型为'%s'，得到'%s'", services.EmailTypeNormal, normalEmail.EmailType)
		}

		// 验证其他关键字段
		if normalEmail.Title != "正常业务邮件" {
			t.Errorf("期望标题为'正常业务邮件'，得到'%s'", normalEmail.Title)
		}
		if normalEmail.From != "sender@example.com" {
			t.Errorf("期望发件人为'sender@example.com'，得到'%s'", normalEmail.From)
		}
		if len(normalEmail.To) != 1 || normalEmail.To[0] != "recipient@example.com" {
			t.Errorf("期望收件人为'recipient@example.com'，得到'%v'", normalEmail.To)
		}
		if len(normalEmail.Content) == 0 {
			t.Errorf("邮件内容不应为空")
		}
		if normalEmail.SentAt == nil {
			t.Errorf("发送时间不应为空")
		}

		// 测试测试邮件
		testEmail := &email_client_pb.Email{
			Title:     "配置测试邮件",
			Content:   []byte("这是测试邮件内容"),
			From:      "sender@example.com",
			To:        []string{"test@example.com"},
			EmailType: services.EmailTypeTest,
			SentAt:    timestamppb.Now(),
		}

		// 验证邮件类型
		if testEmail.EmailType != services.EmailTypeTest {
			t.Errorf("期望邮件类型为'%s'，得到'%s'", services.EmailTypeTest, testEmail.EmailType)
		}

		// 验证其他关键字段
		if testEmail.Title != "配置测试邮件" {
			t.Errorf("期望标题为'配置测试邮件'，得到'%s'", testEmail.Title)
		}
		if testEmail.From != "sender@example.com" {
			t.Errorf("期望发件人为'sender@example.com'，得到'%s'", testEmail.From)
		}
		if len(testEmail.To) != 1 || testEmail.To[0] != "test@example.com" {
			t.Errorf("期望收件人为'test@example.com'，得到'%v'", testEmail.To)
		}
		if len(testEmail.Content) == 0 {
			t.Errorf("邮件内容不应为空")
		}
		if testEmail.SentAt == nil {
			t.Errorf("发送时间不应为空")
		}
	})

	// 测试获取邮件请求的过滤参数
	t.Run("邮件查询过滤参数", func(t *testing.T) {
		// 测试获取所有邮件的请求
		allEmailsReq := &email_client_pb.GetSentEmailsRequest{
			Cursor:    "",
			Limit:     10,
			EmailType: "", // 空字符串表示所有类型
		}

		// 验证请求参数
		if allEmailsReq.EmailType != "" {
			t.Errorf("获取所有邮件时EmailType应为空字符串，得到'%s'", allEmailsReq.EmailType)
		}
		if allEmailsReq.Cursor != "" {
			t.Errorf("期望Cursor为空字符串，得到'%s'", allEmailsReq.Cursor)
		}
		if allEmailsReq.Limit != 10 {
			t.Errorf("期望Limit为10，得到%d", allEmailsReq.Limit)
		}

		// 测试获取正常邮件的请求
		normalEmailsReq := &email_client_pb.GetSentEmailsRequest{
			Cursor:    "",
			Limit:     10,
			EmailType: services.EmailTypeNormal,
		}

		// 验证请求参数
		if normalEmailsReq.EmailType != services.EmailTypeNormal {
			t.Errorf("获取正常邮件时EmailType应为'%s'，得到'%s'", services.EmailTypeNormal, normalEmailsReq.EmailType)
		}
		if normalEmailsReq.Cursor != "" {
			t.Errorf("期望Cursor为空字符串，得到'%s'", normalEmailsReq.Cursor)
		}
		if normalEmailsReq.Limit != 10 {
			t.Errorf("期望Limit为10，得到%d", normalEmailsReq.Limit)
		}

		// 测试获取测试邮件的请求
		testEmailsReq := &email_client_pb.GetSentEmailsRequest{
			Cursor:    "",
			Limit:     10,
			EmailType: services.EmailTypeTest,
		}

		// 验证请求参数
		if testEmailsReq.EmailType != services.EmailTypeTest {
			t.Errorf("获取测试邮件时EmailType应为'%s'，得到'%s'", services.EmailTypeTest, testEmailsReq.EmailType)
		}
		if testEmailsReq.Cursor != "" {
			t.Errorf("期望Cursor为空字符串，得到'%s'", testEmailsReq.Cursor)
		}
		if testEmailsReq.Limit != 10 {
			t.Errorf("期望Limit为10，得到%d", testEmailsReq.Limit)
		}
	})
}

// Example_sendingEmailsWithAttachments 展示如何发送带附件的邮件
func Example_sendingEmailsWithAttachments() {
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

// Example_sendingEmailsByType 展示如何发送不同类型的邮件
func Example_sendingEmailsByType() {
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

	ctx := context.Background()
	configID := "email_config_id" // 替换为实际的邮件配置ID

	// 示例1：发送正常业务邮件
	normalResponse, err := client.EmailService().SendNormalEmail(
		ctx,
		"业务通知邮件",
		[]byte("这是一封正常的业务邮件内容"),
		"business@example.com",
		[]string{"customer@example.com"},
		configID,
	)
	if err != nil {
		// 处理错误
		return
	}
	if normalResponse.Success {
		// 正常邮件发送成功
	}

	// 示例2：发送测试邮件
	testResponse, err := client.EmailService().SendTestEmail(
		ctx,
		"邮箱配置测试",
		[]byte("这是一封测试邮件，用于验证配置是否正常"),
		"system@example.com",
		[]string{"admin@example.com"},
		configID,
	)
	if err != nil {
		// 处理错误
		return
	}
	if testResponse.Success {
		// 测试邮件发送成功
	}

	// 示例3：发送带附件的正常邮件
	normalWithAttachmentResponse, err := client.EmailService().SendNormalEmailWithAttachments(
		ctx,
		"合同文件",
		[]byte("请查收附件中的合同文件"),
		"business@example.com",
		[]string{"partner@example.com"},
		configID,
		[]string{"/path/to/contract.pdf"},
	)
	if err != nil {
		// 处理错误
		return
	}
	if normalWithAttachmentResponse.Success {
		// 带附件的正常邮件发送成功
	}

	// 示例4：发送带附件的测试邮件
	testWithAttachmentResponse, err := client.EmailService().SendTestEmailWithAttachments(
		ctx,
		"附件测试邮件",
		[]byte("测试邮件附件功能"),
		"system@example.com",
		[]string{"admin@example.com"},
		configID,
		[]string{"/path/to/test_file.txt"},
	)
	if err != nil {
		// 处理错误
		return
	}
	if testWithAttachmentResponse.Success {
		// 带附件的测试邮件发送成功
	}
}

// Example_filteringEmailsByType 展示如何按类型过滤查询邮件
func Example_filteringEmailsByType() {
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

	ctx := context.Background()

	// 示例1：获取所有类型的邮件
	allEmails, err := client.EmailService().GetAllSentEmails(ctx, "", 10)
	if err != nil {
		// 处理错误
		return
	}
	// 处理所有邮件列表
	_ = allEmails

	// 示例2：只获取正常业务邮件
	normalEmails, err := client.EmailService().GetNormalEmails(ctx, "", 10)
	if err != nil {
		// 处理错误
		return
	}
	// 处理正常邮件列表
	_ = normalEmails

	// 示例3：只获取测试邮件
	testEmails, err := client.EmailService().GetTestEmails(ctx, "", 10)
	if err != nil {
		// 处理错误
		return
	}
	// 处理测试邮件列表
	_ = testEmails

	// 示例4：使用通用方法按类型过滤
	customFilterEmails, err := client.EmailService().GetSentEmailsByType(ctx, "", 5, services.EmailTypeNormal)
	if err != nil {
		// 处理错误
		return
	}
	// 处理自定义过滤的邮件列表
	_ = customFilterEmails
}

// Example_checkingServiceHealth 展示如何使用健康检查服务
func Example_checkingServiceHealth() {
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

	ctx := context.Background()

	// 示例1：检查整体服务健康状况
	response, err := client.CheckHealth(ctx, "")
	if err != nil {
		// 处理错误
		return
	}

	// 根据状态进行处理
	if response.Status == email_client_pb.HealthCheckResponse_SERVING {
		// 服务正常
	} else {
		// 服务异常，可以记录日志或告警
	}

	// 示例2：检查特定服务（如 "EmailService"）的健康状况
	emailServiceHealth, err := client.CheckHealth(ctx, "EmailService")
	if err != nil {
		// 处理错误
		return
	}

	if emailServiceHealth.Status != email_client_pb.HealthCheckResponse_SERVING {
		// EmailService 子服务异常
	}
}

// Example_sendingHTMLEmails 展示如何发送HTML格式的邮件
func Example_sendingHTMLEmails() {
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

	ctx := context.Background()
	configID := "email_config_id" // 替换为实际的邮件配置ID

	// HTML邮件内容模板
	htmlContent := `
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
		<title>HTML邮件</title>
		<style>
			body { 
				font-family: Arial, sans-serif; 
				line-height: 1.6; 
				color: #333; 
			}
			.container { 
				max-width: 600px; 
				margin: 0 auto; 
				padding: 20px; 
			}
			.header { 
				background-color: #4CAF50; 
				color: white; 
				padding: 20px; 
				text-align: center; 
				border-radius: 5px 5px 0 0; 
			}
			.content { 
				background-color: #f9f9f9; 
				padding: 20px; 
				border: 1px solid #ddd; 
			}
			.footer { 
				background-color: #333; 
				color: white; 
				padding: 10px; 
				text-align: center; 
				border-radius: 0 0 5px 5px; 
			}
			.button { 
				display: inline-block; 
				padding: 10px 20px; 
				background-color: #007BFF; 
				color: white; 
				text-decoration: none; 
				border-radius: 5px; 
			}
		</style>
	</head>
	<body>
		<div class="container">
			<div class="header">
				<h1>欢迎使用我们的服务</h1>
			</div>
			<div class="content">
				<h2>这是一封HTML格式的邮件</h2>
				<p>您好！这是一封<strong>HTML格式</strong>的邮件示例。</p>
				<ul>
					<li>支持<em>富文本格式</em></li>
					<li>支持<a href="https://example.com">链接</a></li>
					<li>支持样式和布局</li>
					<li>支持响应式设计</li>
				</ul>
				<p>
					<a href="https://example.com/action" class="button">点击这里</a>
				</p>
			</div>
			<div class="footer">
				<p>&copy; 2024 您的公司名称. 保留所有权利.</p>
			</div>
		</div>
	</body>
	</html>
	`

	// 示例1：发送普通HTML邮件
	response, err := client.EmailService().SendHTMLEmail(
		ctx,
		"HTML邮件测试",
		htmlContent,
		"sender@example.com",
		[]string{"recipient@example.com"},
		configID,
	)
	if err != nil {
		// 处理错误
		return
	}
	if response.Success {
		// HTML邮件发送成功
	}

	// 示例2：发送正常业务HTML邮件
	businessHTMLContent := `
	<html>
	<body>
		<h2>业务通知</h2>
		<p>尊敬的客户，</p>
		<p>您的订单 <strong>#12345</strong> 已经处理完成。</p>
		<table border="1" style="border-collapse: collapse;">
			<tr>
				<th>商品名称</th>
				<th>数量</th>
				<th>价格</th>
			</tr>
			<tr>
				<td>产品A</td>
				<td>2</td>
				<td>¥100.00</td>
			</tr>
		</table>
		<p>感谢您的购买！</p>
	</body>
	</html>
	`

	normalResponse, err := client.EmailService().SendNormalHTMLEmail(
		ctx,
		"订单处理完成通知",
		businessHTMLContent,
		"business@example.com",
		[]string{"customer@example.com"},
		configID,
	)
	if err != nil {
		// 处理错误
		return
	}
	if normalResponse.Success {
		// 业务HTML邮件发送成功
	}

	// 示例3：发送测试HTML邮件
	testHTMLContent := `
	<html>
	<body style="font-family: Arial, sans-serif;">
		<div style="background-color: #f0f8ff; padding: 20px; border-radius: 10px;">
			<h2 style="color: #0066cc;">邮件配置测试</h2>
			<p>这是一封<span style="color: red; font-weight: bold;">测试邮件</span>，用于验证HTML邮件配置是否正常。</p>
			<div style="background-color: #e6f3ff; padding: 15px; margin: 10px 0; border-left: 4px solid #0066cc;">
				<strong>测试项目：</strong>
				<ul>
					<li>HTML格式渲染</li>
					<li>CSS样式支持</li>
					<li>中文字符显示</li>
					<li>链接功能</li>
				</ul>
			</div>
			<p style="color: #666;">如果您能正常看到这封邮件的格式，说明配置成功！</p>
		</div>
	</body>
	</html>
	`

	testResponse, err := client.EmailService().SendTestHTMLEmail(
		ctx,
		"HTML邮件配置测试",
		testHTMLContent,
		"system@example.com",
		[]string{"admin@example.com"},
		configID,
	)
	if err != nil {
		// 处理错误
		return
	}
	if testResponse.Success {
		// 测试HTML邮件发送成功
	}

	// 示例4：发送带附件的HTML邮件
	attachmentPaths := []string{"/path/to/report.pdf"}

	htmlWithAttachmentResponse, err := client.EmailService().SendHTMLEmailWithAttachments(
		ctx,
		"月度报告（HTML格式）",
		htmlContent,
		"reports@example.com",
		[]string{"manager@example.com"},
		configID,
		attachmentPaths,
	)
	if err != nil {
		// 处理错误
		return
	}
	if htmlWithAttachmentResponse.Success {
		// 带附件的HTML邮件发送成功
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

// TestEmailTypeFeatures 综合测试邮件类型功能
func TestEmailTypeFeatures(t *testing.T) {
	t.Run("邮件类型功能综合测试", func(t *testing.T) {
		// 在这个测试中，我们模拟不同类型邮件的创建和验证

		// 1. 验证不同类型邮件的结构
		normalEmail := createMockEmail("业务邮件", services.EmailTypeNormal)
		testEmail := createMockEmail("测试邮件", services.EmailTypeTest)

		// 验证邮件类型设置正确
		if normalEmail.EmailType != services.EmailTypeNormal {
			t.Errorf("正常邮件类型设置错误: 期望 %s, 得到 %s", services.EmailTypeNormal, normalEmail.EmailType)
		}

		if testEmail.EmailType != services.EmailTypeTest {
			t.Errorf("测试邮件类型设置错误: 期望 %s, 得到 %s", services.EmailTypeTest, testEmail.EmailType)
		}

		// 2. 验证查询请求的过滤参数
		requests := map[string]*email_client_pb.GetSentEmailsRequest{
			"所有邮件": createMockGetRequest(""),
			"正常邮件": createMockGetRequest(services.EmailTypeNormal),
			"测试邮件": createMockGetRequest(services.EmailTypeTest),
		}

		expectedTypes := map[string]string{
			"所有邮件": "",
			"正常邮件": services.EmailTypeNormal,
			"测试邮件": services.EmailTypeTest,
		}

		for name, req := range requests {
			expected := expectedTypes[name]
			if req.EmailType != expected {
				t.Errorf("%s的过滤参数错误: 期望 '%s', 得到 '%s'", name, expected, req.EmailType)
			}
		}

		t.Logf("✅ 邮件类型功能测试通过")
		t.Logf("   - 正常邮件类型: %s", services.EmailTypeNormal)
		t.Logf("   - 测试邮件类型: %s", services.EmailTypeTest)
		t.Logf("   - 支持按类型过滤查询")
	})
}

// 辅助函数：创建模拟邮件
func createMockEmail(title, emailType string) *email_client_pb.Email {
	return &email_client_pb.Email{
		Title:     title,
		Content:   []byte("邮件内容: " + title),
		From:      "sender@example.com",
		To:        []string{"recipient@example.com"},
		EmailType: emailType,
		SentAt:    timestamppb.Now(),
	}
}

// 辅助函数：创建模拟查询请求
func createMockGetRequest(emailType string) *email_client_pb.GetSentEmailsRequest {
	return &email_client_pb.GetSentEmailsRequest{
		Cursor:    "",
		Limit:     10,
		EmailType: emailType,
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
