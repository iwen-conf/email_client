package emailclient

import (
	"context"
	"fmt"
	"google.golang.org/grpc/connectivity"
	"time"

	"github.com/iwen-conf/email_client/proto/email_client_pb" // 确保这个导入路径在你的 go.mod 中是正确的
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultConnectTimeout = 10 * time.Second // 默认连接超时
)

// EmailServiceClient 封装了与邮件服务交互的 gRPC 客户端。
type EmailServiceClient struct {
	client          email_client_pb.EmailServiceClient
	conn            *grpc.ClientConn
	requestTimeout  time.Duration
	defaultPageSize int32
}

// NewEmailServiceClient 创建一个新的 EmailServiceClient 实例，并连接到指定的 gRPC 服务地址。
// 使用 grpc.DialContext 建立连接，并默认阻塞直到连接成功或超时。
func NewEmailServiceClient(grpcAddress string, requestTimeout time.Duration, defaultPageSize int32) (*EmailServiceClient, error) {
	if grpcAddress == "" {
		return nil, fmt.Errorf("gRPC 服务地址不能为空")
	}

	// 建立 gRPC 连接，使用 DialContext
	conn, err := grpc.NewClient(grpcAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("连接邮件服务失败 (%s): %w", grpcAddress, err)
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
	// 创建 EmailServiceClient 存根
	grpcClient := email_client_pb.NewEmailServiceClient(conn)

	return &EmailServiceClient{
		client:          grpcClient,
		conn:            conn,
		requestTimeout:  requestTimeout,
		defaultPageSize: defaultPageSize,
	}, nil
}

// Close 关闭与邮件服务的 gRPC 连接。
func (c *EmailServiceClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// GetClient 返回底层的 email_client_pb.EmailServiceClient 存根。
// 允许直接调用 gRPC 方法。
func (c *EmailServiceClient) GetClient() email_client_pb.EmailServiceClient {
	return c.client
}

// SetRequestTimeout 设置默认的请求超时时间。
func (c *EmailServiceClient) SetRequestTimeout(timeout time.Duration) {
	c.requestTimeout = timeout
}

// SetDefaultPageSize 设置默认的分页大小。
func (c *EmailServiceClient) SetDefaultPageSize(size int32) {
	c.defaultPageSize = size
}

// --- SDK 方法封装 ---

// GetSentEmails 调用 gRPC 服务获取已发送邮件列表。
func (c *EmailServiceClient) GetSentEmails(ctx context.Context, req *email_client_pb.GetSentEmailsRequest) (*email_client_pb.GetSentEmailsResponse, error) {
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

	return c.client.GetSentEmails(ctx, req)
}

// SendEmail 调用 gRPC 服务发送单封邮件。
func (c *EmailServiceClient) SendEmail(ctx context.Context, req *email_client_pb.SendEmailRequest) (*email_client_pb.SendEmailResponse, error) {
	// 应用请求超时
	if c.requestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.requestTimeout)
		defer cancel()
	}
	return c.client.SendEmail(ctx, req)
}

// SendEmails 调用 gRPC 服务批量发送多封邮件。
func (c *EmailServiceClient) SendEmails(ctx context.Context, req *email_client_pb.SendEmailsRequest) (*email_client_pb.SendEmailsResponse, error) {
	// 应用请求超时
	if c.requestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.requestTimeout)
		defer cancel()
	}
	return c.client.SendEmails(ctx, req)
}
