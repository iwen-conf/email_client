package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/iwen-conf/email_client/client"
	"github.com/iwen-conf/email_client/proto/email_client_pb"
)

func main() {
	// 定义命令行参数
	serverAddr := flag.String("server", "localhost:50051", "gRPC 服务器地址")
	timeout := flag.Duration("timeout", 10*time.Second, "请求超时时间")
	pageSize := flag.Int("pagesize", 20, "默认分页大小")
	debug := flag.Bool("debug", false, "是否启用调试日志")

	// 断路器相关参数
	enableCircuitBreaker := flag.Bool("circuit", false, "是否启用断路器")
	failureThreshold := flag.Int("failures", 5, "断路器触发的失败阈值")
	resetTimeout := flag.Duration("reset", 10*time.Second, "断路器重置时间")

	// 健康检查相关参数
	healthCheckInterval := flag.Duration("health", 30*time.Second, "健康检查间隔，0表示禁用")

	// 重试相关参数
	maxRetries := flag.Int("retries", 3, "最大重试次数")
	retryDelay := flag.Duration("retrydelay", 500*time.Millisecond, "重试延迟")

	// 执行操作标志
	listConfigs := flag.Bool("list-configs", false, "列出所有邮件配置")
	listEmails := flag.Bool("list-emails", false, "列出已发送的邮件")

	// 解析命令行参数
	flag.Parse()

	fmt.Println("gRPC Email Client v0.1.0")
	fmt.Printf("连接服务: %s\n", *serverAddr)

	// 准备客户端选项
	var options []client.Option

	// 配置健康检查
	if *healthCheckInterval > 0 {
		options = append(options, client.EnableHealthCheck(*healthCheckInterval))
		fmt.Printf("已启用健康检查，间隔: %v\n", *healthCheckInterval)
	} else {
		options = append(options, client.DisableHealthCheck())
		fmt.Println("已禁用健康检查")
	}

	// 配置断路器
	if *enableCircuitBreaker {
		options = append(options, client.WithCircuitBreakerConfig(client.CircuitBreakerConfig{
			FailureThreshold:    *failureThreshold,
			ResetTimeout:        *resetTimeout,
			HalfOpenMaxRequests: 1,
		}))
		fmt.Printf("已启用断路器，失败阈值: %d, 重置时间: %v\n", *failureThreshold, *resetTimeout)
	} else {
		options = append(options, client.DisableCircuitBreaker())
		fmt.Println("已禁用断路器")
	}

	// 配置重试
	options = append(options, client.WithRetryConfig(client.RetryConfig{
		MaxRetries:  *maxRetries,
		RetryDelay:  *retryDelay,
		RetryPolicy: client.ExponentialBackoff,
	}))
	fmt.Printf("已配置重试机制，最大重试次数: %d, 重试延迟: %v\n", *maxRetries, *retryDelay)

	// 创建邮件客户端
	emailClient, err := client.NewEmailClient(*serverAddr, *timeout, int32(*pageSize), *debug, options...)
	if err != nil {
		log.Fatalf("创建邮件客户端失败: %v", err)
	}
	defer emailClient.Close()

	fmt.Println("邮件客户端已成功连接")
	fmt.Println("提供以下服务:")
	fmt.Println("- 邮件服务 (EmailService)")
	fmt.Println("- 配置服务 (ConfigService)")

	// 执行示例操作
	ctx := context.Background()

	if *listConfigs {
		// 示例：列出所有配置
		fmt.Println("\n===== 获取配置列表 =====")
		// 创建 ListConfigsRequest
		req := &email_client_pb.ListConfigsRequest{
			Page:     1,
			PageSize: int32(*pageSize),
		}
		configs, err := emailClient.ConfigService().ListConfigs(ctx, req)
		if err != nil {
			fmt.Printf("获取配置列表失败: %v\n", err)
		} else {
			fmt.Printf("共找到 %d 个配置:\n", configs.Total)
			for i, config := range configs.Configs {
				fmt.Printf("%d. %s (ID: %s) - %s:%d\n",
					i+1, config.Name, config.Id, config.Server, config.Port)
			}
		}
	}

	if *listEmails {
		// 示例：列出发送的邮件
		fmt.Println("\n===== 获取发送邮件列表 =====")
		// 创建 GetSentEmailsRequest
		req := &email_client_pb.GetSentEmailsRequest{
			Page:     1,
			PageSize: int32(*pageSize),
		}
		emails, err := emailClient.EmailService().GetSentEmails(ctx, req)
		if err != nil {
			fmt.Printf("获取邮件列表失败: %v\n", err)
		} else {
			fmt.Printf("共找到 %d 封邮件:\n", emails.Total)
			for i, email := range emails.Emails {
				fmt.Printf("%d. 标题: %s, 发件人: %s, 收件人: %v\n",
					i+1, email.Title, email.From, email.To)
			}
		}
	}

	// 如果没有执行任何操作，展示创建配置和发送邮件的示例代码
	if !*listConfigs && !*listEmails {
		fmt.Println("\n===== 示例操作 =====")
		fmt.Println("创建配置示例:")
		fmt.Println(`
		config := &email_client_pb.EmailConfig{
			Protocol: email_client_pb.EmailConfig_SMTP,
			Server:   "smtp.example.com",
			Port:     587,
			UseSsl:   true,
			Username: "user@example.com",
			Password: "password",
			Name:     "示例配置",
		}
		req := &email_client_pb.CreateConfigRequest{
			Config: config,
		}
		resp, err := emailClient.ConfigService().CreateConfig(ctx, req)
		`)

		fmt.Println("\n发送邮件示例:")
		fmt.Println(`
		email := &email_client_pb.Email{
			Title:   "测试邮件",
			Content: []byte("这是一封测试邮件"),
			From:    "sender@example.com",
			To:      []string{"recipient@example.com"},
			SentAt:  timestamppb.Now(),
		}
		req := &email_client_pb.SendEmailRequest{
			Email:    email,
			ConfigId: "配置ID",
		}
		resp, err := emailClient.EmailService().SendEmail(ctx, req)
		`)
	}

	// 等待信号以优雅退出
	fmt.Println("\n按 Ctrl+C 退出")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	fmt.Println("正在关闭客户端...")
}
