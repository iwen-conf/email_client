package main

import (
	"fmt"
	"log"
	"time"

	// 导入新的 SDK 包
	"github.com/iwen-conf/email_client/pkg/emailclient" // 确保这个路径与你的 go.mod 模块路径匹配
)

func main() {
	// gRPC 服务地址
	grpcAddress := "localhost:50051"

	// 创建 EmailServiceClient
	emailClient, err := emailclient.NewEmailServiceClient(grpcAddress, 15*time.Second, 10)
	if err != nil {
		log.Fatalf("无法创建 EmailServiceClient: %v", err)
	}
	defer emailClient.Close() // 确保关闭连接

	fmt.Println("成功连接到邮件服务!")

	// 创建 ConfigServiceClient
	configClient, err := emailclient.NewConfigServiceClient(grpcAddress, 10*time.Second, 5)
	if err != nil {
		log.Fatalf("无法创建 ConfigServiceClient: %v", err)
	}
	defer configClient.Close() // 确保关闭连接

	fmt.Println("成功连接到邮件配置服务!")

	// --- 在这里可以使用 emailClient 和 configClient 调用 gRPC 方法 ---
	// 例如获取底层 gRPC 客户端存根:
	// pbEmailClient := emailClient.GetClient()
	// pbConfigClient := configClient.GetClient()

	// 调用示例 (需要替换为实际的 gRPC 方法和请求)
	/*
	   ctx, cancel := context.WithTimeout(context.Background(), emailClient.requestTimeout) // 使用配置的超时或自定义
	   defer cancel()

	   resp, err := pbEmailClient.SomeMethod(ctx, &pb.SomeRequest{...})
	   if err != nil {
	       log.Fatalf("调用失败: %v", err)
	   }
	   fmt.Printf("调用成功: %v\n", resp)
	*/

}
