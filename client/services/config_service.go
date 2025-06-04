package services

import (
	"context"
	"time"

	"github.com/iwen-conf/email_client/proto/email_client_pb"
	"google.golang.org/grpc"
)

// ConfigServiceClient 封装了与邮件配置服务交互的 gRPC 客户端。
type ConfigServiceClient struct {
	client          email_client_pb.EmailConfigServiceClient
	conn            *grpc.ClientConn
	requestTimeout  time.Duration
	defaultPageSize int32
	debug           bool
}

// NewConfigServiceClient 创建一个使用已存在连接的 ConfigServiceClient 实例。
func NewConfigServiceClient(conn *grpc.ClientConn, requestTimeout time.Duration, defaultPageSize int32, debug bool) *ConfigServiceClient {
	// 创建 gRPC 存根
	grpcClient := email_client_pb.NewEmailConfigServiceClient(conn)

	return &ConfigServiceClient{
		client:          grpcClient,
		conn:            conn,
		requestTimeout:  requestTimeout,
		defaultPageSize: defaultPageSize,
		debug:           debug,
	}
}

// GetClient 返回底层的 email_client_pb.EmailConfigServiceClient 存根。
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

	// 如果请求中未设置 Limit，可以使用默认值
	if req.GetLimit() == 0 {
		req.Limit = c.defaultPageSize
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
