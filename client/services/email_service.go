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

// EmailType 定义邮件类型常量
const (
	EmailTypeNormal = "normal" // 正常业务邮件
	EmailTypeTest   = "test"   // 测试配置邮件
)

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

	// 如果请求中未设置 Limit，可以使用默认值
	if req.GetLimit() == 0 {
		req.Limit = c.defaultPageSize
	}

	return c.client.GetSentEmails(ctx, req)
}

// GetSentEmailsByType 按邮件类型获取已发送邮件列表（便捷方法）
func (c *EmailServiceClient) GetSentEmailsByType(ctx context.Context, cursor string, limit int32, emailType string) (*email_client_pb.GetSentEmailsResponse, error) {
	// 应用请求超时
	if c.requestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.requestTimeout)
		defer cancel()
	}

	// 设置默认分页大小
	if limit == 0 {
		limit = c.defaultPageSize
	}

	req := &email_client_pb.GetSentEmailsRequest{
		Cursor:    cursor,
		Limit:     limit,
		EmailType: emailType, // 新增的邮件类型过滤
	}

	return c.client.GetSentEmails(ctx, req)
}

// GetAllSentEmails 获取所有类型的已发送邮件（便捷方法）
func (c *EmailServiceClient) GetAllSentEmails(ctx context.Context, cursor string, limit int32) (*email_client_pb.GetSentEmailsResponse, error) {
	return c.GetSentEmailsByType(ctx, cursor, limit, "") // 空字符串表示所有类型
}

// GetNormalEmails 获取正常业务邮件（便捷方法）
func (c *EmailServiceClient) GetNormalEmails(ctx context.Context, cursor string, limit int32) (*email_client_pb.GetSentEmailsResponse, error) {
	return c.GetSentEmailsByType(ctx, cursor, limit, EmailTypeNormal)
}

// GetTestEmails 获取测试邮件（便捷方法）
func (c *EmailServiceClient) GetTestEmails(ctx context.Context, cursor string, limit int32) (*email_client_pb.GetSentEmailsResponse, error) {
	return c.GetSentEmailsByType(ctx, cursor, limit, EmailTypeTest)
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

// SendNormalEmail 发送正常业务邮件（便捷方法）
func (c *EmailServiceClient) SendNormalEmail(
	ctx context.Context,
	title string,
	content []byte,
	from string,
	to []string,
	configID string,
) (*email_client_pb.SendEmailResponse, error) {
	return c.sendEmailWithType(ctx, title, content, from, to, configID, EmailTypeNormal, nil)
}

// SendTestEmail 发送测试邮件（便捷方法）
func (c *EmailServiceClient) SendTestEmail(
	ctx context.Context,
	title string,
	content []byte,
	from string,
	to []string,
	configID string,
) (*email_client_pb.SendEmailResponse, error) {
	return c.sendEmailWithType(ctx, title, content, from, to, configID, EmailTypeTest, nil)
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
	return c.sendEmailWithType(ctx, title, content, from, to, configID, EmailTypeNormal, attachmentPaths)
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

// SendNormalEmailWithAttachments 发送带附件的正常业务邮件（便捷方法）
func (c *EmailServiceClient) SendNormalEmailWithAttachments(
	ctx context.Context,
	title string,
	content []byte,
	from string,
	to []string,
	configID string,
	attachmentPaths []string,
) (*email_client_pb.SendEmailResponse, error) {
	return c.sendEmailWithType(ctx, title, content, from, to, configID, EmailTypeNormal, attachmentPaths)
}

// SendTestEmailWithAttachments 发送带附件的测试邮件（便捷方法）
func (c *EmailServiceClient) SendTestEmailWithAttachments(
	ctx context.Context,
	title string,
	content []byte,
	from string,
	to []string,
	configID string,
	attachmentPaths []string,
) (*email_client_pb.SendEmailResponse, error) {
	return c.sendEmailWithType(ctx, title, content, from, to, configID, EmailTypeTest, attachmentPaths)
}

// SendHTMLEmail 发送HTML格式邮件（便捷方法）
func (c *EmailServiceClient) SendHTMLEmail(
	ctx context.Context,
	title string,
	htmlContent string,
	from string,
	to []string,
	configID string,
) (*email_client_pb.SendEmailResponse, error) {
	return c.sendEmailWithType(ctx, title, []byte(htmlContent), from, to, configID, EmailTypeNormal, nil)
}

// SendNormalHTMLEmail 发送正常业务HTML邮件（便捷方法）
func (c *EmailServiceClient) SendNormalHTMLEmail(
	ctx context.Context,
	title string,
	htmlContent string,
	from string,
	to []string,
	configID string,
) (*email_client_pb.SendEmailResponse, error) {
	return c.sendEmailWithType(ctx, title, []byte(htmlContent), from, to, configID, EmailTypeNormal, nil)
}

// SendTestHTMLEmail 发送测试HTML邮件（便捷方法）
func (c *EmailServiceClient) SendTestHTMLEmail(
	ctx context.Context,
	title string,
	htmlContent string,
	from string,
	to []string,
	configID string,
) (*email_client_pb.SendEmailResponse, error) {
	return c.sendEmailWithType(ctx, title, []byte(htmlContent), from, to, configID, EmailTypeTest, nil)
}

// SendHTMLEmailWithAttachments 发送带附件的HTML邮件（便捷方法）
func (c *EmailServiceClient) SendHTMLEmailWithAttachments(
	ctx context.Context,
	title string,
	htmlContent string,
	from string,
	to []string,
	configID string,
	attachmentPaths []string,
) (*email_client_pb.SendEmailResponse, error) {
	return c.sendEmailWithType(ctx, title, []byte(htmlContent), from, to, configID, EmailTypeNormal, attachmentPaths)
}

// SendNormalHTMLEmailWithAttachments 发送带附件的正常业务HTML邮件（便捷方法）
func (c *EmailServiceClient) SendNormalHTMLEmailWithAttachments(
	ctx context.Context,
	title string,
	htmlContent string,
	from string,
	to []string,
	configID string,
	attachmentPaths []string,
) (*email_client_pb.SendEmailResponse, error) {
	return c.sendEmailWithType(ctx, title, []byte(htmlContent), from, to, configID, EmailTypeNormal, attachmentPaths)
}

// SendTestHTMLEmailWithAttachments 发送带附件的测试HTML邮件（便捷方法）
func (c *EmailServiceClient) SendTestHTMLEmailWithAttachments(
	ctx context.Context,
	title string,
	htmlContent string,
	from string,
	to []string,
	configID string,
	attachmentPaths []string,
) (*email_client_pb.SendEmailResponse, error) {
	return c.sendEmailWithType(ctx, title, []byte(htmlContent), from, to, configID, EmailTypeTest, attachmentPaths)
}

// sendEmailWithType 内部方法：发送指定类型的邮件
func (c *EmailServiceClient) sendEmailWithType(
	ctx context.Context,
	title string,
	content []byte,
	from string,
	to []string,
	configID string,
	emailType string,
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
		Title:     title,
		Content:   content,
		From:      from,
		To:        to,
		EmailType: emailType, // 设置邮件类型
	}

	// 处理附件
	if len(attachmentPaths) > 0 {
		attachments, err := c.loadAttachments(attachmentPaths)
		if err != nil {
			return nil, err
		}
		email.Attachments = attachments
	}

	// 创建并发送请求
	req := &email_client_pb.SendEmailRequest{
		Email:    email,
		ConfigId: configID,
	}

	return c.client.SendEmail(ctx, req)
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
