package client

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc/connectivity"

	"github.com/iwen-conf/email_client/proto/email_client_pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// EmailClient 是一个高级客户端，封装了与邮件服务和配置服务的交互。
// 它管理一个共享的 gRPC 连接。
type EmailClient struct {
	conn            *grpc.ClientConn
	emailService    *EmailServiceClient
	configService   *ConfigServiceClient
	requestTimeout  time.Duration
	defaultPageSize int32
}

// NewEmailClient 创建一个新的 EmailClient 实例。
// 它会建立一个到指定 gRPC 服务地址的连接，并初始化底层的服务客户端。
func NewEmailClient(grpcAddress string, requestTimeout time.Duration, defaultPageSize int32) (*EmailClient, error) {
	if grpcAddress == "" {
		log.Println("[ERROR] NewEmailClient: gRPC 服务地址不能为空")
		return nil, fmt.Errorf("gRPC 服务地址不能为空")
	}

	log.Printf("[INFO] NewEmailClient: 正在尝试连接统一 gRPC 服务: %s", grpcAddress)

	// 建立共享的 gRPC 连接
	conn, err := grpc.NewClient(grpcAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Printf("[ERROR] NewEmailClient: 连接 gRPC 服务失败 (%s): %v", grpcAddress, err)
		return nil, fmt.Errorf("连接 gRPC 服务失败 (%s): %w", grpcAddress, err)
	}

	// 主动连接并等待 Ready 状态
	conn.Connect()
	log.Printf("[INFO] NewEmailClient: 正在等待连接变为 Ready 状态 (%s)...", grpcAddress)

	ctx, cancel := context.WithTimeout(context.Background(), defaultConnectTimeout)
	defer cancel()

	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			log.Printf("[INFO] NewEmailClient: 成功连接到 gRPC 服务 (%s)", grpcAddress)
			break // 成功连接
		}
		if !conn.WaitForStateChange(ctx, state) {
			// 关闭尝试失败的连接
			conn.Close()
			errMsg := fmt.Sprintf("等待连接状态变化超时或被取消 (%s)", grpcAddress)
			log.Printf("[ERROR] NewEmailClient: %s", errMsg)
			return nil, fmt.Errorf(errMsg)
		}
		currentState := conn.GetState()
		log.Printf("[DEBUG] NewEmailClient: 连接状态变化 (%s): %v -> %v", grpcAddress, state, currentState)
		// 检查是否已经进入失败状态
		if currentState == connectivity.TransientFailure || currentState == connectivity.Shutdown {
			// 关闭尝试失败的连接
			conn.Close()
			errMsg := fmt.Sprintf("连接失败，当前状态: %v (%s)", currentState, grpcAddress)
			log.Printf("[ERROR] NewEmailClient: %s", errMsg)
			return nil, fmt.Errorf(errMsg)
		}
	}

	// 创建 gRPC 存根
	emailGrpcClient := email_client_pb.NewEmailServiceClient(conn)
	configGrpcClient := email_client_pb.NewEmailConfigServiceClient(conn)

	log.Printf("[INFO] NewEmailClient: 已创建 EmailService 和 ConfigService 客户端 (%s)", grpcAddress)

	// 创建内部的服务客户端实例
	emailService := &EmailServiceClient{
		client:          emailGrpcClient,
		conn:            conn, // 注意：这里传递了共享连接，但 EmailServiceClient 的 Close 不应关闭它
		requestTimeout:  requestTimeout,
		defaultPageSize: defaultPageSize,
	}
	configService := &ConfigServiceClient{
		client:          configGrpcClient,
		conn:            conn, // 注意：这里传递了共享连接，但 ConfigServiceClient 的 Close 不应关闭它
		requestTimeout:  requestTimeout,
		defaultPageSize: defaultPageSize,
	}

	return &EmailClient{
		conn:            conn,
		emailService:    emailService,
		configService:   configService,
		requestTimeout:  requestTimeout,
		defaultPageSize: defaultPageSize,
	}, nil
}

// Close 关闭 EmailClient 管理的共享 gRPC 连接。
// 注意：调用此方法后，通过 EmailService() 和 ConfigService() 获取的客户端实例也将失效。
func (c *EmailClient) Close() error {
	if c.conn != nil {
		log.Printf("[INFO] EmailClient.Close: 正在关闭共享 gRPC 连接: %s", c.conn.Target())
		// 只有 EmailClient 的 Close 才真正关闭连接
		err := c.conn.Close()
		// 将内部客户端的 conn 设置为 nil，防止它们的 Close 方法尝试再次关闭
		if c.emailService != nil {
			c.emailService.conn = nil
		}
		if c.configService != nil {
			c.configService.conn = nil
		}
		return err
	}
	return nil
}

// EmailService 返回封装好的 EmailServiceClient 实例。
func (c *EmailClient) EmailService() *EmailServiceClient {
	return c.emailService
}

// ConfigService 返回封装好的 ConfigServiceClient 实例。
func (c *EmailClient) ConfigService() *ConfigServiceClient {
	return c.configService
}

// SetRequestTimeout 设置所有内部服务客户端的默认请求超时时间。
func (c *EmailClient) SetRequestTimeout(timeout time.Duration) {
	c.requestTimeout = timeout
	if c.emailService != nil {
		c.emailService.SetRequestTimeout(timeout)
	}
	if c.configService != nil {
		c.configService.SetRequestTimeout(timeout)
	}
}

// SetDefaultPageSize 设置所有内部服务客户端的默认分页大小。
func (c *EmailClient) SetDefaultPageSize(size int32) {
	c.defaultPageSize = size
	if c.emailService != nil {
		c.emailService.SetDefaultPageSize(size)
	}
	if c.configService != nil {
		c.configService.SetDefaultPageSize(size)
	}
}
