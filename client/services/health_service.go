package services

import (
	"context"
	"time"

	"github.com/iwen-conf/email_client/proto/email_client_pb"
	"google.golang.org/grpc"
)

// HealthServiceClient 封装了与健康检查服务相关的所有操作
type HealthServiceClient struct {
	grpcClient     email_client_pb.HealthServiceClient
	requestTimeout time.Duration
	debug          bool
}

// NewHealthServiceClient 创建一个新的健康检查服务客户端
func NewHealthServiceClient(conn *grpc.ClientConn, requestTimeout time.Duration, debug bool) *HealthServiceClient {
	return &HealthServiceClient{
		grpcClient:     email_client_pb.NewHealthServiceClient(conn),
		requestTimeout: requestTimeout,
		debug:          debug,
	}
}

// Check 调用健康检查服务的Check方法
// serviceName 是要检查的服务名称，如果为空，则检查整体服务器健康状况。
func (c *HealthServiceClient) Check(ctx context.Context, serviceName string) (*email_client_pb.HealthCheckResponse, error) {
	req := &email_client_pb.HealthCheckRequest{
		Service: serviceName,
	}

	// 设置请求超时
	ctx, cancel := context.WithTimeout(ctx, c.requestTimeout)
	defer cancel()

	return c.grpcClient.Check(ctx, req)
}
 