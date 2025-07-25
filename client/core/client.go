package core

import (
	"context"
	"log"
	"time"

	"github.com/iwen-conf/email_client/client/conn"
	"github.com/iwen-conf/email_client/client/services"
	"github.com/iwen-conf/email_client/proto/email_client_pb"
)

// EmailClient 是一个高级客户端，封装了与邮件服务和配置服务的交互。
type EmailClient struct {
	connManager     *conn.Manager
	emailService    *services.EmailServiceClient
	configService   *services.ConfigServiceClient
	healthService   *services.HealthServiceClient
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

	// 创建连接管理器选项
	managerOpts := []conn.ManagerOption{
		conn.WithHealthCheck(options.enableHealthCheck, options.healthCheckInterval),
	}

	// 添加TLS选项
	if options.enableTLS {
		managerOpts = append(managerOpts, conn.WithTLS(options.tlsConfig))
		if debug {
			log.Printf("[INFO] NewEmailClient: 启用TLS安全连接")
		}
	}

	// 创建连接管理器
	connManager, err := conn.NewManager(grpcAddress, options.minConnectTimeout, debug, managerOpts...)
	if err != nil {
		return nil, err
	}

	// 创建内部的服务客户端实例
	emailService := services.NewEmailServiceClient(connManager.GetConn(), requestTimeout, defaultPageSize, debug)
	configService := services.NewConfigServiceClient(connManager.GetConn(), requestTimeout, defaultPageSize, debug)
	healthService := services.NewHealthServiceClient(connManager.GetConn(), requestTimeout, debug)

	if debug {
		log.Printf("[INFO] NewEmailClient: 成功创建所有服务客户端 (Email, Config, Health)")
	}

	return &EmailClient{
		connManager:     connManager,
		emailService:    emailService,
		configService:   configService,
		healthService:   healthService,
		requestTimeout:  requestTimeout,
		defaultPageSize: defaultPageSize,
		debug:           debug,
	}, nil
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

// HealthService 返回健康检查服务的客户端实例
func (c *EmailClient) HealthService() *services.HealthServiceClient {
	return c.healthService
}

// CheckHealth 是一个便捷方法，用于检查整体服务的健康状况。
// serviceName 是要检查的服务名称，如果为空，则检查整体服务器健康状况。
func (c *EmailClient) CheckHealth(ctx context.Context, serviceName string) (*email_client_pb.HealthCheckResponse, error) {
	if c.healthService == nil {
		return nil, ErrHealthServiceNotInitialized
	}
	return c.healthService.Check(ctx, serviceName)
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
	if c.healthService != nil {
		// 假设 HealthServiceClient 也有 SetRequestTimeout 方法
		// c.healthService.SetRequestTimeout(timeout)
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

// UpdateTLSConfig 更新TLS配置并重新连接
func (c *EmailClient) UpdateTLSConfig(config conn.TLSConfig) error {
	manager := c.GetConnManager()
	manager.UpdateTLSConfig(config)

	// 重新连接以应用新配置
	ctx, cancel := context.WithTimeout(context.Background(), c.requestTimeout)
	defer cancel()

	return manager.Reconnect(ctx, "")
}
