package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/iwen-conf/email_client/client"
	"github.com/iwen-conf/email_client/proto/email_client_pb"
)

func main() {
	// gRPC 服务地址
	grpcAddress := "localhost:50051"
	// 统一的请求超时和默认分页大小 (可以根据需要调整)
	requestTimeout := 15 * time.Second
	defaultPageSize := int32(10)

	// 创建统一的 EmailClient
	emailClient, err := client.NewEmailClient(grpcAddress, requestTimeout, defaultPageSize)
	if err != nil {
		log.Fatalf("无法创建 EmailClient: %v", err)
	}
	defer emailClient.Close() // 确保关闭共享连接

	fmt.Println("成功连接到 gRPC 服务!")

	// --- 通过 EmailClient 获取各个服务的客户端实例 ---
	emailService := emailClient.EmailService()
	configService := emailClient.ConfigService()

	// 可以在这里分别使用 emailService 和 configService 调用方法
	fmt.Printf("获取到的 Email Service 类型: %T\n", emailService)
	fmt.Printf("获取到的 Config Service 类型: %T\n", configService)

	// 获取底层 gRPC 客户端存根 (如果需要直接调用)
	// pbEmailClient := emailService.GetClient()
	// pbConfigClient := configService.GetClient()

	// 调用示例 (需要替换为实际的 gRPC 方法和请求)
	fmt.Println("\n--- 运行示例调用 ---")
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout) // 使用 EmailClient 配置的超时
	defer cancel()

	// 使用 emailService 调用邮件服务方法 (示例)
	sendReq := &email_client_pb.SendEmailRequest{
		ConfigId: "default", // 使用的邮件配置ID
		Email: &email_client_pb.Email{ // 嵌套的 Email 消息
			Title:   "来自 Go SDK 的测试邮件",                       // 邮件标题 (title -> Title)
			Content: []byte("这是一封通过统一 EmailClient 发送的测试邮件。"), // 邮件内容 (content -> Content, type bytes)
			To:      []string{"test@example.com"},            // 收件人列表 (to -> To)
			// From: "sender@example.com", // 可选：发件人 (from -> From)
		},
	}
	fmt.Printf("尝试发送邮件: %+v\n", sendReq)
	sendResp, err := emailService.SendEmail(ctx, sendReq)
	if err != nil {
		// 对于示例，我们只打印错误，不终止程序
		log.Printf("发送邮件失败: %v\n", err)
	} else {
		fmt.Printf("发送邮件调用成功 (不代表邮件已发出): %v\n", sendResp)
	}

	// 使用 configService 调用配置服务方法 (示例)
	listReq := &email_client_pb.ListConfigsRequest{
		Page: 1, // 页码 (page -> Page)
		// PageSize 会使用 EmailClient 中设置的默认值 (page_size -> PageSize)
		// 可以覆盖: PageSize: 5,
	}
	fmt.Printf("\n尝试获取配置列表: %+v\n", listReq)
	listResp, err := configService.ListConfigs(ctx, listReq)
	if err != nil {
		// 对于示例，我们只打印错误，不终止程序
		log.Printf("获取配置列表失败: %v\n", err)
	} else {
		fmt.Printf("获取配置列表成功，数量: %d\n", len(listResp.Configs))
		// 可以选择性打印配置信息
		// for _, cfg := range listResp.Configs {
		//  fmt.Printf(" - Config ID: %s, Host: %s\n", cfg.Id, cfg.Host)
		// }
	}

}
