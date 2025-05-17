package services

import (
	"context"
	"os"
	"path/filepath"
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

// SendEmailWithAttachments 发送带附件的邮件
// title: 邮件标题
// content: 邮件内容
// from: 发件人地址
// to: 收件人地址列表
// configID: 邮件配置ID
// attachmentPaths: 附件文件路径列表
func (c *EmailServiceClient) SendEmailWithAttachments(
	ctx context.Context,
	title string,
	content []byte,
	from string,
	to []string,
	configID string,
	attachmentPaths []string,
) (*email_client_pb.SendEmailResponse, error) {
	// 应用请求超时
	if c.requestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.requestTimeout)
		defer cancel()
	}

	// 创建邮件
	email := &email_client_pb.Email{
		Title:   title,
		Content: content,
		From:    from,
		To:      to,
	}

	// 处理附件
	attachments, err := c.loadAttachments(attachmentPaths)
	if err != nil {
		return nil, err
	}
	email.Attachments = attachments

	// 创建并发送请求
	req := &email_client_pb.SendEmailRequest{
		Email:    email,
		ConfigId: configID,
	}

	return c.client.SendEmail(ctx, req)
}

// SendEmailWithAttachment 发送带单个附件的邮件（便捷方法）
func (c *EmailServiceClient) SendEmailWithAttachment(
	ctx context.Context,
	title string,
	content []byte,
	from string,
	to []string,
	configID string,
	attachmentPath string,
) (*email_client_pb.SendEmailResponse, error) {
	return c.SendEmailWithAttachments(ctx, title, content, from, to, configID, []string{attachmentPath})
}

// loadAttachments 从文件路径加载附件
func (c *EmailServiceClient) loadAttachments(filePaths []string) ([]*email_client_pb.Attachment, error) {
	attachments := make([]*email_client_pb.Attachment, 0, len(filePaths))

	for _, path := range filePaths {
		// 读取文件内容
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		// 获取文件信息
		fileInfo, err := os.Stat(path)
		if err != nil {
			return nil, err
		}

		// 创建附件对象
		attachment := &email_client_pb.Attachment{
			Filename:    filepath.Base(path),
			Content:     content,
			ContentType: getContentType(path), // 简单的MIME类型检测
			Size:        fileInfo.Size(),
		}

		attachments = append(attachments, attachment)
	}

	return attachments, nil
}

// getContentType 根据文件扩展名返回MIME类型
func getContentType(filename string) string {
	ext := filepath.Ext(filename)
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain"
	case ".html":
		return "text/html"
	case ".doc", ".docx":
		return "application/msword"
	case ".xls", ".xlsx":
		return "application/vnd.ms-excel"
	case ".zip":
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}
