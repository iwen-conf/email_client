package client

import (
	"context"
	"fmt"
	"google.golang.org/grpc/connectivity"
	"time"

	"github.com/iwen-conf/email_client/proto/email_client_pb" // 确保这个导入路径在你的 go.mod 中是正确的
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ConfigServiceClient 封装了与邮件配置服务交互的 gRPC 客户端。
type ConfigServiceClient struct {
	client          email_client_pb.EmailConfigServiceClient
	conn            *grpc.ClientConn
	requestTimeout  time.Duration
	defaultPageSize int32
}

// NewConfigServiceClient 创建一个新的 ConfigServiceClient 实例，并连接到指定的 gRPC 服务地址。
// 使用 grpc.DialContext 建立连接，并默认阻塞直到连接成功或超时。
func NewConfigServiceClient(grpcAddress string, requestTimeout time.Duration, defaultPageSize int32) (*ConfigServiceClient, error) {
	if grpcAddress == "" {
		return nil, fmt.Errorf("gRPC 服务地址不能为空")
	}

	// 建立 gRPC 连接，使用 DialContext
	conn, err := grpc.NewClient(grpcAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("连接邮件配置服务失败 (%s): %w", grpcAddress, err)
	}
	// 主动连接
	conn.Connect()
	// 设置连接超时
	ctx, cancel := context.WithTimeout(context.Background(), defaultConnectTimeout)
	defer cancel()

	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			break // 成功连接
		}
		if !conn.WaitForStateChange(ctx, state) {
			return nil, fmt.Errorf("等待连接状态变化超时或被取消")
		}
		// 检查是否已经进入失败状态
		if conn.GetState() == connectivity.TransientFailure || conn.GetState() == connectivity.Shutdown {
			return nil, fmt.Errorf("连接失败，当前状态: %v", conn.GetState())
		}
	}
	// 创建 EmailConfigServiceClient 存根
	grpcClient := email_client_pb.NewEmailConfigServiceClient(conn)

	return &ConfigServiceClient{
		client:          grpcClient,
		conn:            conn,
		requestTimeout:  requestTimeout,
		defaultPageSize: defaultPageSize,
	}, nil
}

// Close 关闭与邮件配置服务的 gRPC 连接。
func (c *ConfigServiceClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// GetClient 返回底层的 email_client_pb.EmailConfigServiceClient 存根。
// 允许直接调用 gRPC 方法。
func (c *ConfigServiceClient) GetClient() email_client_pb.EmailConfigServiceClient {
	return c.client
}

// SetRequestTimeout 设置默认的请求超时时间。
func (c *ConfigServiceClient) SetRequestTimeout(timeout time.Duration) {
	c.requestTimeout = timeout
}

// SetDefaultPageSize 设置默认的分页大小。
func (c *ConfigServiceClient) SetDefaultPageSize(size int32) {
	c.defaultPageSize = size
}

// --- SDK 方法封装 ---

// CreateConfig 调用 gRPC 服务创建新的邮件配置。
func (c *ConfigServiceClient) CreateConfig(ctx context.Context, req *email_client_pb.CreateConfigRequest) (*email_client_pb.ConfigResponse, error) {
	// 应用请求超时
	if c.requestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.requestTimeout)
		defer cancel()
	}
	return c.client.CreateConfig(ctx, req)
}

// GetConfig 调用 gRPC 服务根据 ID 获取指定邮件配置。
func (c *ConfigServiceClient) GetConfig(ctx context.Context, req *email_client_pb.GetConfigRequest) (*email_client_pb.ConfigResponse, error) {
	// 应用请求超时
	if c.requestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.requestTimeout)
		defer cancel()
	}
	return c.client.GetConfig(ctx, req)
}

// UpdateConfig 调用 gRPC 服务更新指定邮件配置。
func (c *ConfigServiceClient) UpdateConfig(ctx context.Context, req *email_client_pb.UpdateConfigRequest) (*email_client_pb.ConfigResponse, error) {
	// 应用请求超时
	if c.requestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.requestTimeout)
		defer cancel()
	}
	return c.client.UpdateConfig(ctx, req)
}

// DeleteConfig 调用 gRPC 服务删除指定邮件配置。
func (c *ConfigServiceClient) DeleteConfig(ctx context.Context, req *email_client_pb.DeleteConfigRequest) (*email_client_pb.DeleteConfigResponse, error) {
	// 应用请求超时
	if c.requestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.requestTimeout)
		defer cancel()
	}
	return c.client.DeleteConfig(ctx, req)
}

// ListConfigs 调用 gRPC 服务获取所有邮件配置列表。
func (c *ConfigServiceClient) ListConfigs(ctx context.Context, req *email_client_pb.ListConfigsRequest) (*email_client_pb.ListConfigsResponse, error) {
	// 应用请求超时
	if c.requestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.requestTimeout)
		defer cancel()
	}

	// 如果请求中未设置 PageSize，可以使用默认值
	if req.GetPageSize() == 0 {
		req.PageSize = c.defaultPageSize
	}

	return c.client.ListConfigs(ctx, req)
}

// TestConfig 调用 gRPC 服务测试邮件配置是否可用。
func (c *ConfigServiceClient) TestConfig(ctx context.Context, req *email_client_pb.TestConfigRequest) (*email_client_pb.TestConfigResponse, error) {
	// 应用请求超时
	if c.requestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.requestTimeout)
		defer cancel()
	}
	return c.client.TestConfig(ctx, req)
}
