package services

import (
	"context"
	"time"

	"github.com/iwen-conf/email_client/proto/email_client_pb"
	"google.golang.org/grpc"
)

// EmailServiceClient 封装了与邮件服务交互的 gRPC 客户端。
type EmailServiceClient struct {
	client          email_client_pb.EmailServiceClient
	conn            *grpc.ClientConn
	requestTimeout  time.Duration
	defaultPageSize int32
	debug           bool
}

// NewEmailServiceClient 创建一个使用已存在连接的 EmailServiceClient 实例。
func NewEmailServiceClient(conn *grpc.ClientConn, requestTimeout time.Duration, defaultPageSize int32, debug bool) *EmailServiceClient {
	// 创建 gRPC 存根
	grpcClient := email_client_pb.NewEmailServiceClient(conn)

	return &EmailServiceClient{
		client:          grpcClient,
		conn:            conn,
		requestTimeout:  requestTimeout,
		defaultPageSize: defaultPageSize,
		debug:           debug,
	}
}

// GetClient 返回底层的 email_client_pb.EmailServiceClient 存根。
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
