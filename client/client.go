package client

import (
	"context"
	"fmt"
	"log"
	"sync"
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
	debug           bool

	// 添加健康检查和重连相关字段
	healthChecker   *HealthChecker
	healthCheckLock sync.Mutex
	options         clientOptions
	connectionMutex sync.Mutex
	target          string // 记录连接目标，用于重连
}

// NewEmailClient 创建一个新的 EmailClient 实例。
// debug 参数控制是否打印 INFO 和 DEBUG 级别的日志。
// 增加可选的配置选项参数
func NewEmailClient(grpcAddress string, requestTimeout time.Duration, defaultPageSize int32, debug bool, opts ...Option) (*EmailClient, error) {
	if grpcAddress == "" {
		log.Println("[ERROR] NewEmailClient: gRPC 服务地址不能为空")
		return nil, fmt.Errorf("gRPC 服务地址不能为空")
	}

	// 处理选项
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}

	if debug {
		log.Printf("[INFO] NewEmailClient: 正在尝试连接统一 gRPC 服务: %s", grpcAddress)
	}

	client := &EmailClient{
		requestTimeout:  requestTimeout,
		defaultPageSize: defaultPageSize,
		debug:           debug,
		options:         options,
		target:          grpcAddress,
	}

	// 建立连接
	ctx, cancel := context.WithTimeout(context.Background(), options.minConnectTimeout)
	defer cancel()

	if err := client.connect(ctx); err != nil {
		return nil, err
	}

	// 如果启用了健康检查，创建并启动健康检查器
	if options.enableHealthCheck && options.healthCheckInterval > 0 {
		client.startHealthCheck()
	}

	return client, nil
}

// connect 建立 gRPC 连接
func (c *EmailClient) connect(ctx context.Context) error {
	c.connectionMutex.Lock()
	defer c.connectionMutex.Unlock()

	// 建立共享的 gRPC 连接
	conn, err := grpc.NewClient(c.target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Printf("[ERROR] NewEmailClient: 连接 gRPC 服务失败 (%s): %v", c.target, err)
		return fmt.Errorf("连接 gRPC 服务失败 (%s): %w", c.target, err)
	}

	// 主动连接并等待 Ready 状态
	conn.Connect()
	if c.debug {
		log.Printf("[INFO] NewEmailClient: 正在等待连接变为 Ready 状态 (%s)...", c.target)
	}

	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			if c.debug {
				log.Printf("[INFO] NewEmailClient: 成功连接到 gRPC 服务 (%s)", c.target)
			}
			break // 成功连接
		}
		if !conn.WaitForStateChange(ctx, state) {
			conn.Close()
			errMsg := fmt.Sprintf("等待连接状态变化超时或被取消 (%s)", c.target)
			log.Printf("[ERROR] NewEmailClient: %s", errMsg)
			return fmt.Errorf("%s", errMsg)
		}
		currentState := conn.GetState()
		if c.debug {
			log.Printf("[DEBUG] NewEmailClient: 连接状态变化 (%s): %v -> %v", c.target, state, currentState)
		}
		if currentState == connectivity.TransientFailure || currentState == connectivity.Shutdown {
			conn.Close()
			errMsg := fmt.Sprintf("连接失败，当前状态: %v (%s)", currentState, c.target)
			log.Printf("[ERROR] NewEmailClient: %s", errMsg)
			return fmt.Errorf("%s", errMsg)
		}
	}

	// 创建 gRPC 存根
	emailGrpcClient := email_client_pb.NewEmailServiceClient(conn)
	configGrpcClient := email_client_pb.NewEmailConfigServiceClient(conn)

	if c.debug {
		log.Printf("[INFO] NewEmailClient: 已创建 EmailService 和 ConfigService 客户端 (%s)", c.target)
	}

	// 创建内部的服务客户端实例
	emailService := &EmailServiceClient{
		client:          emailGrpcClient,
		conn:            conn,
		requestTimeout:  c.requestTimeout,
		defaultPageSize: c.defaultPageSize,
		debug:           c.debug,
	}
	configService := &ConfigServiceClient{
		client:          configGrpcClient,
		conn:            conn,
		requestTimeout:  c.requestTimeout,
		defaultPageSize: c.defaultPageSize,
		debug:           c.debug,
	}

	// 设置客户端内部状态
	c.conn = conn
	c.emailService = emailService
	c.configService = configService

	return nil
}

// reconnect 重新建立 gRPC 连接
// 用于健康检查器在检测到连接断开时尝试重连
func (c *EmailClient) reconnect(ctx context.Context, target string) error {
	if target != "" {
		c.target = target
	}
	return c.connect(ctx)
}

// startHealthCheck 启动健康检查
func (c *EmailClient) startHealthCheck() {
	c.healthCheckLock.Lock()
	defer c.healthCheckLock.Unlock()

	if c.healthChecker != nil {
		c.healthChecker.Stop()
	}

	c.healthChecker = NewHealthChecker(c, c.options.healthCheckInterval, c.debug)
	c.healthChecker.Start()
}

// stopHealthCheck 停止健康检查
func (c *EmailClient) stopHealthCheck() {
	c.healthCheckLock.Lock()
	defer c.healthCheckLock.Unlock()

	if c.healthChecker != nil {
		c.healthChecker.Stop()
		c.healthChecker = nil
	}
}

// Close 关闭 EmailClient 管理的共享 gRPC 连接。
// 注意：调用此方法后，通过 EmailService() 和 ConfigService() 获取的客户端实例也将失效。
func (c *EmailClient) Close() error {
	// 停止健康检查
	c.stopHealthCheck()

	c.connectionMutex.Lock()
	defer c.connectionMutex.Unlock()

	if c.conn != nil {
		if c.debug {
			log.Printf("[INFO] EmailClient.Close: 正在关闭共享 gRPC 连接: %s", c.conn.Target())
		}
		err := c.conn.Close()
		if c.emailService != nil {
			c.emailService.conn = nil
		}
		if c.configService != nil {
			c.configService.conn = nil
		}
		c.conn = nil
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
