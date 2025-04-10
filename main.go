package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/iwen-conf/email_client/client"
	"github.com/iwen-conf/email_client/proto/email_client_pb"
)

func main() {
	// --- 配置参数 ---
	grpcAddress := flag.String("addr", "localhost:50051", "gRPC 服务地址")
	requestTimeoutSec := flag.Int("timeout", 15, "默认请求超时时间 (秒)")
	defaultPageSize := flag.Int("pagesize", 10, "默认分页大小")
	configIDToGet := flag.String("get-config-id", "default", "要获取的配置ID (用于 GetConfig 示例)")
	configIDToSend := flag.String("send-config-id", "default", "发送邮件时使用的配置ID (用于 SendEmail 示例)")
	testEmailRecipient := flag.String("to", "test@example.com", "测试邮件接收者")
	debugMode := flag.Bool("debug", false, "是否启用调试日志") // 改为 -debug 标志和 debugMode 变量

	flag.Parse() // 解析命令行标志

	// 将秒转换为 time.Duration
	requestTimeout := time.Duration(*requestTimeoutSec) * time.Second

	fmt.Printf("连接到 gRPC 服务: %s, 超时: %s, 默认分页大小: %d, 调试模式: %t\n",
		*grpcAddress, requestTimeout, *defaultPageSize, *debugMode) // 更新打印信息

	// 创建统一的 EmailClient, 传递 debug 标志
	emailClient, err := client.NewEmailClient(*grpcAddress, requestTimeout, int32(*defaultPageSize), *debugMode)
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
		ConfigId: *configIDToSend, // 使用的邮件配置ID
		Email: &email_client_pb.Email{ // 嵌套的 Email 消息
			Title:   "来自 Go SDK 的测试邮件",                       // 邮件标题 (title -> Title)
			Content: []byte("这是一封通过统一 EmailClient 发送的测试邮件。"), // 邮件内容 (content -> Content, type bytes)
			To:      []string{*testEmailRecipient},           // 收件人列表 (to -> To)
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

	// 3. 示例: 获取单个配置
	fmt.Println("\n3. 示例: 获取单个配置")
	if *configIDToGet != "" {
		getReq := &email_client_pb.GetConfigRequest{
			Id: *configIDToGet, // 使用命令行指定的配置ID
		}
		fmt.Printf("尝试获取配置请求: %+v\n", getReq)
		getResp, err := configService.GetConfig(ctx, getReq)
		if err != nil {
			log.Printf("获取配置 '%s' 失败: %v\n", *configIDToGet, err)
		} else if getResp.Success {
			fmt.Printf("获取配置 '%s' 成功:\n", *configIDToGet)
			fmt.Printf("  ID: %s\n", getResp.Config.Id)
			fmt.Printf("  名称: %s\n", getResp.Config.Name)
			fmt.Printf("  服务器: %s:%d\n", getResp.Config.Server, getResp.Config.Port)
			fmt.Printf("  协议: %s\n", getResp.Config.Protocol)
			fmt.Printf("  用户名: %s\n", getResp.Config.Username)
			fmt.Printf("  使用SSL: %t\n", getResp.Config.UseSsl)
			// 注意: 时间戳需要检查是否为 nil
			if getResp.Config.CreatedAt != nil {
				fmt.Printf("  创建时间: %s\n", getResp.Config.CreatedAt.AsTime().Local())
			}
			if getResp.Config.UpdatedAt != nil {
				fmt.Printf("  更新时间: %s\n", getResp.Config.UpdatedAt.AsTime().Local())
			}
		} else {
			fmt.Printf("获取配置 '%s' 失败: %s\n", *configIDToGet, getResp.Message)
		}
	} else {
		fmt.Println("未指定 -get-config-id, 跳过 GetConfig 示例。")
	}

	fmt.Println("\n--- 示例调用结束 ---")
}
