package core

import (
	"log"
	"time"

	"github.com/iwen-conf/email_client/client/conn"
	"github.com/iwen-conf/email_client/client/services"
)

// EmailClient 是一个高级客户端，封装了与邮件服务和配置服务的交互。
type EmailClient struct {
	connManager     *conn.Manager
	emailService    *services.EmailServiceClient
	configService   *services.ConfigServiceClient
	requestTimeout  time.Duration
	defaultPageSize int32
	debug           bool
}

// NewEmailClient 创建一个新的 EmailClient 实例。
func NewEmailClient(grpcAddress string, requestTimeout time.Duration, defaultPageSize int32, debug bool, opts ...Option) (*EmailClient, error) {
	if grpcAddress == "" {
		log.Println("[ERROR] NewEmailClient: gRPC 服务地址不能为空")
		return nil, ErrEmptyGrpcAddress
	}

	// 处理选项
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}

	if debug {
		log.Printf("[INFO] NewEmailClient: 正在尝试连接统一 gRPC 服务: %s", grpcAddress)
	}

	// 创建连接管理器
	connManager, err := conn.NewManager(grpcAddress, options.minConnectTimeout, debug, conn.WithHealthCheck(options.enableHealthCheck, options.healthCheckInterval))
	if err != nil {
		return nil, err
	}

	// 创建内部的服务客户端实例
	emailService := services.NewEmailServiceClient(connManager.GetConn(), requestTimeout, defaultPageSize, debug)
	configService := services.NewConfigServiceClient(connManager.GetConn(), requestTimeout, defaultPageSize, debug)

	client := &EmailClient{
		connManager:     connManager,
		emailService:    emailService,
		configService:   configService,
		requestTimeout:  requestTimeout,
		defaultPageSize: defaultPageSize,
		debug:           debug,
	}

	return client, nil
}

// Close 关闭 EmailClient 管理的共享 gRPC 连接。
func (c *EmailClient) Close() error {
	if c.debug {
		log.Printf("[INFO] EmailClient.Close: 正在关闭共享 gRPC 连接")
	}
	return c.connManager.Close()
}

// EmailService 返回封装好的 EmailServiceClient 实例。
func (c *EmailClient) EmailService() *services.EmailServiceClient {
	return c.emailService
}

// ConfigService 返回封装好的 ConfigServiceClient 实例。
func (c *EmailClient) ConfigService() *services.ConfigServiceClient {
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

// GetConnManager 返回底层的连接管理器
func (c *EmailClient) GetConnManager() *conn.Manager {
	return c.connManager
}
