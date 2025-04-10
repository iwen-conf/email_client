package client

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc/connectivity"

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
	debug           bool // 改为 debug 标志
}

// NewEmailServiceClient 创建一个新的 EmailServiceClient 实例，并连接到指定的 gRPC 服务地址。
// 使用 grpc.DialContext 建立连接，并默认阻塞直到连接成功或超时。
func NewEmailServiceClient(grpcAddress string, requestTimeout time.Duration, defaultPageSize int32, debug bool) (*EmailServiceClient, error) {
	if grpcAddress == "" {
		log.Println("[ERROR] NewEmailServiceClient: gRPC 服务地址不能为空")
		return nil, fmt.Errorf("gRPC 服务地址不能为空")
	}

	if debug {
		log.Printf("[INFO] NewEmailServiceClient: 正在尝试连接邮件服务: %s", grpcAddress)
	}

	// 建立 gRPC 连接
	conn, err := grpc.NewClient(grpcAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Printf("[ERROR] NewEmailServiceClient: 连接邮件服务失败 (%s): %v", grpcAddress, err)
		return nil, fmt.Errorf("连接邮件服务失败 (%s): %w", grpcAddress, err)
	}

	// 主动连接并等待 Ready 状态
	conn.Connect()
	if debug {
		log.Printf("[INFO] NewEmailServiceClient: 正在等待连接变为 Ready 状态 (%s)...", grpcAddress)
	}
	// 使用包级别的 defaultConnectTimeout
	ctx, cancel := context.WithTimeout(context.Background(), defaultConnectTimeout)
	defer cancel()

	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			if debug {
				log.Printf("[INFO] NewEmailServiceClient: 成功连接到邮件服务 (%s)", grpcAddress)
			}
			break // 成功连接
		}
		if !conn.WaitForStateChange(ctx, state) {
			conn.Close() // 关闭尝试失败的连接
			errMsg := fmt.Sprintf("等待邮件服务连接状态变化超时或被取消 (%s)", grpcAddress)
			log.Printf("[ERROR] NewEmailServiceClient: %s", errMsg)
			return nil, fmt.Errorf("%s", errMsg)
		}
		currentState := conn.GetState()
		if debug {
			log.Printf("[DEBUG] NewEmailServiceClient: 连接状态变化 (%s): %v -> %v", grpcAddress, state, currentState)
		}
		// 检查是否已经进入失败状态
		if currentState == connectivity.TransientFailure || currentState == connectivity.Shutdown {
			conn.Close() // 关闭尝试失败的连接
			errMsg := fmt.Sprintf("邮件服务连接失败，当前状态: %v (%s)", currentState, grpcAddress)
			log.Printf("[ERROR] NewEmailServiceClient: %s", errMsg)
			return nil, fmt.Errorf("%s", errMsg)
		}
	}

	// 创建 EmailServiceClient 存根
	grpcClient := email_client_pb.NewEmailServiceClient(conn)
	if debug {
		log.Printf("[INFO] NewEmailServiceClient: 已创建 EmailService 客户端 (%s)", grpcAddress)
	}

	return &EmailServiceClient{
		client:          grpcClient,
		conn:            conn,
		requestTimeout:  requestTimeout,
		defaultPageSize: defaultPageSize,
		debug:           debug, // 存储 debug
	}, nil
}

// Close 关闭与邮件服务的 gRPC 连接。
func (c *EmailServiceClient) Close() error {
	if c.conn != nil {
		if c.debug {
			log.Printf("[INFO] EmailServiceClient.Close: 正在关闭邮件服务连接: %s", c.conn.Target())
		}
		return c.conn.Close()
	}
	if c.debug {
		log.Println("[INFO] EmailServiceClient.Close: 连接已关闭或由 EmailClient 管理，无需操作")
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
