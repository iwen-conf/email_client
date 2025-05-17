# gRPC Email Client

一个功能齐全的 gRPC 邮件客户端库，为邮件服务和配置服务提供高级接口。作为外部库导入使用，当前版本 v0.0.1。

## 主要特性

- **统一连接管理**：使用单个连接同时访问邮件服务和配置服务
- **连接池管理**：高效管理多个 gRPC 连接，提升并发性能
- **结构化日志**：支持不同日志级别、格式和输出方式的日志系统
- **速率限制**：基于令牌桶算法的API访问速率限制
- **TLS安全连接**：支持证书验证和加密传输
- **健康检查**：自动检测连接健康状态并进行自动重连
- **请求重试机制**：支持可配置的失败重试策略
- **断路器模式**：防止系统雪崩，自动中断连接到不健康的服务
- **性能指标收集**：监控请求执行情况和性能指标
- **选项模式配置**：灵活的客户端配置系统
- **模块化架构**：清晰的职责分离，便于维护和扩展

## 安装

```bash
go get github.com/iwen-conf/email_client
```

## 快速开始

### 创建客户端

```go
import (
    "context"
    "time"
    "github.com/iwen-conf/email_client/client"
)

func main() {
    // 创建带默认选项的客户端
    emailClient, err := client.NewEmailClient(
        "localhost:50051",
        10*time.Second, // 请求超时
        20,             // 默认分页大小
        true,           // 启用调试日志
    )
    if err != nil {
        panic(err)
    }
    defer emailClient.Close()
    
    // 使用客户端...
}
```

### 使用高级选项

```go
// 启用健康检查
options := []client.Option{
    client.EnableHealthCheck(30*time.Second),
}

// 启用断路器
options = append(options, client.WithCircuitBreakerConfig(client.CircuitBreakerConfig{
    FailureThreshold:    5,
    ResetTimeout:        10*time.Second,
    HalfOpenMaxRequests: 1,
}))

// 配置重试策略
options = append(options, client.WithRetryConfig(client.RetryConfig{
    MaxRetries:  3,
    RetryDelay:  500*time.Millisecond,
    RetryPolicy: client.ExponentialBackoff,
}))

// 配置速率限制
options = append(options, client.WithRateLimiterConfig(client.RateLimiterConfig{
    RequestsPerSecond: 20.0,  // 每秒最大请求数
    MaxBurst:          30.0,  // 最大突发请求数
    WaitTimeout:       100*time.Millisecond, // 等待令牌的超时时间
}))

// 配置TLS安全连接
options = append(options, client.WithTLSConfig(client.TLSConfig{
    Enabled:            true,                // 启用TLS
    ServerName:         "email.example.com", // 服务器名称
    CertFile:           "/path/to/cert.pem", // 客户端证书
    KeyFile:            "/path/to/key.pem",  // 客户端密钥
    CAFile:             "/path/to/ca.pem",   // CA证书
    InsecureSkipVerify: false,               // 是否跳过证书验证
}))

// 创建带选项的客户端
emailClient, err := client.NewEmailClient(
    "localhost:50051", 
    10*time.Second, 
    20, 
    true, 
    options...,
)
```

## 使用示例

### 邮件服务

```go
// 获取已发送邮件列表
ctx := context.Background()
req := &email_client_pb.GetSentEmailsRequest{
    Page:     1,
    PageSize: 20,
}
emails, err := emailClient.EmailService().GetSentEmails(ctx, req)
if err != nil {
    // 处理错误
}

// 发送邮件
email := &email_client_pb.Email{
    Title:   "测试邮件",
    Content: []byte("这是一封测试邮件"),
    From:    "sender@example.com",
    To:      []string{"recipient@example.com"},
    SentAt:  timestamppb.Now(),
}
sendReq := &email_client_pb.SendEmailRequest{
    Email:    email,
    ConfigId: "config123",
}
resp, err := emailClient.EmailService().SendEmail(ctx, sendReq)

// 发送带附件的邮件（便捷方法）
ctx := context.Background()
title := "带附件的邮件"
content := []byte("这是一封包含附件的邮件")
from := "sender@example.com"
to := []string{"recipient@example.com"}
configID := "config123"

// 发送单个附件
attachmentPath := "/path/to/document.pdf"
resp, err := emailClient.EmailService().SendEmailWithAttachment(
    ctx, title, content, from, to, configID, attachmentPath,
)

// 发送多个附件
attachmentPaths := []string{
    "/path/to/document.pdf",
    "/path/to/image.jpg",
    "/path/to/spreadsheet.xlsx",
}
resp, err = emailClient.EmailService().SendEmailWithAttachments(
    ctx, title, content, from, to, configID, attachmentPaths,
)
```

### 配置服务

```go
// 创建邮件配置
config := &email_client_pb.EmailConfig{
    Protocol: email_client_pb.EmailConfig_SMTP,
    Server:   "smtp.example.com",
    Port:     587,
    UseSsl:   true,
    Username: "user@example.com",
    Password: "password",
    Name:     "示例配置",
}
createReq := &email_client_pb.CreateConfigRequest{
    Config: config,
}
createResp, err := emailClient.ConfigService().CreateConfig(ctx, createReq)

// 获取配置列表
listReq := &email_client_pb.ListConfigsRequest{
    Page:     1,
    PageSize: 20,
}
configs, err := emailClient.ConfigService().ListConfigs(ctx, listReq)
```

## 高级功能说明

### TLS安全连接

使用TLS加密可保护通信安全，支持证书验证和加密传输。

```go
import (
    "github.com/iwen-conf/email_client/client/conn"
)

// 创建自定义TLS配置
tlsConfig := conn.TLSConfig{
    Enabled:            true,                // 启用TLS
    ServerName:         "email.example.com", // 用于证书验证的服务器名称
    CertFile:           "/path/to/cert.pem", // 客户端证书文件路径
    KeyFile:            "/path/to/key.pem",  // 客户端密钥文件路径
    CAFile:             "/path/to/ca.pem",   // CA证书文件路径
    InsecureSkipVerify: false,               // 是否跳过证书验证(不推荐在生产环境中设为true)
}

// 使用TLS配置创建连接管理器
manager, err := conn.NewManager("localhost:50051", 10*time.Second, true, 
    conn.WithTLS(tlsConfig),
    conn.WithHealthCheck(true, 30*time.Second),
)
if err != nil {
    panic(err)
}
defer manager.Close()

// 使用连接发起请求
// ...

// 动态更新TLS配置
newTLSConfig := conn.TLSConfig{
    Enabled:            true,
    ServerName:         "new.example.com",
    InsecureSkipVerify: false,
}
manager.UpdateTLSConfig(newTLSConfig)

// 重新连接以应用新配置
ctx := context.Background()
if err := manager.Reconnect(ctx, ""); err != nil {
    // 处理错误
}
```

### 连接池管理

连接池可以高效管理多个gRPC连接，提高并发性能和资源利用率。

```go
import (
    "github.com/iwen-conf/email_client/client/conn"
)

// 创建自定义连接池配置
poolConfig := conn.DefaultPoolConfig()
poolConfig.InitialSize = 5
poolConfig.MaxSize = 20
poolConfig.MinIdle = 2
poolConfig.MaxIdle = 10*time.Minute
poolConfig.HealthCheckInterval = 60*time.Second

// 创建连接工厂函数
factory := func() (*grpc.ClientConn, error) {
    return grpc.NewClient("localhost:50051",
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
}

// 创建连接池
pool, err := conn.NewConnectionPool("localhost:50051", factory, poolConfig)
if err != nil {
    panic(err)
}
defer pool.Close()

// 从连接池获取连接
ctx := context.Background()
connection, err := pool.Get(ctx)
if err != nil {
    panic(err)
}
defer connection.Release() // 使用完后释放回连接池
```

### 结构化日志

客户端内置了结构化日志系统，支持不同级别、格式和输出方式。

```go
import (
    "os"
    "github.com/iwen-conf/email_client/client/logger"
)

// 创建日志记录器
log := logger.NewStandardLogger()

// 设置日志级别
log.SetLevel(logger.InfoLevel)

// 设置日志输出到文件
file, _ := os.OpenFile("email_client.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
log.SetOutput(file)

// 使用JSON格式
log.SetFormatter(&logger.JSONFormatter{TimeFormat: time.RFC3339})

// 使用日志
log.Info("客户端初始化成功")
log.WithField("grpc_address", "localhost:50051").Info("连接服务器")
log.WithRequestID("req-123").WithField("user", "admin").Info("处理请求")

// 条件日志
if log.GetLevel() <= logger.DebugLevel {
    // 只有在调试级别时才会执行这些昂贵的操作
    log.Debug("详细调试信息")
}

// 带错误信息的日志
err := someOperation()
if err != nil {
    log.WithError(err).Error("操作失败")
}
```

### 速率限制

速率限制器可防止API过度使用，保护服务器资源并确保公平访问。

```go
import (
    "context"
    "github.com/iwen-conf/email_client/client/middleware"
)

// 创建速率限制器
config := client.DefaultRateLimiterConfig()
config.RequestsPerSecond = 50.0  // 每秒50个请求
config.MaxBurst = 100.0          // 最大突发请求数
config.WaitTimeout = 200*time.Millisecond  // 等待超时时间

rateLimiter := client.NewRateLimiter(config, true)

// 在执行请求前检查速率限制
ctx := context.Background()
err := rateLimiter.Wait(ctx)
if err != nil {
    // 处理速率限制错误
    if limitErr, ok := err.(*client.RateLimitExceededError); ok {
        log.Printf("速率限制超出: %.2f 请求/秒, %s", limitErr.RequestsPerSecond, limitErr.Message)
        return
    }
}

// 正常执行请求
// ...

// 动态调整速率限制
rateLimiter.SetRate(100.0)  // 提高限制到每秒100请求
```

### 健康检查

健康检查系统会定期检查与服务器的连接状态，并在连接断开时自动重连。

```go
// 启用健康检查，30秒间隔
options = append(options, client.EnableHealthCheck(30*time.Second))

// 禁用健康检查
options = append(options, client.DisableHealthCheck())
```

### 重试机制

客户端内置了请求重试机制，对短暂的服务故障具有弹性。

```go
// 配置重试策略
options = append(options, client.WithRetryConfig(client.RetryConfig{
    MaxRetries:  3,               // 最大重试次数
    RetryDelay:  500*time.Millisecond, // 初始重试延迟
    RetryPolicy: client.ExponentialBackoff, // 重试策略
}))
```

### 断路器模式

断路器可以防止系统在面对服务持续故障时过载。

```go
// 启用断路器
options = append(options, client.WithCircuitBreakerConfig(client.CircuitBreakerConfig{
    FailureThreshold:    5,               // 连续失败次数阈值
    ResetTimeout:        10*time.Second, // 断路器重置时间
    HalfOpenMaxRequests: 1,               // 半开状态最大请求数
}))

// 禁用断路器
options = append(options, client.DisableCircuitBreaker())
```

## 项目结构

- **client/**: 客户端包
  - **entry.go**: 包入口点，重新导出API保持兼容性
  - **core/**: 核心客户端功能
    - **client.go**: 主客户端实现
    - **options.go**: 客户端选项系统
    - **errors.go**: 错误定义
  - **services/**: 服务客户端实现
    - **email_service.go**: 邮件服务客户端
    - **config_service.go**: 配置服务客户端
  - **conn/**: 连接管理
    - **manager.go**: 连接管理器
    - **pool.go**: 连接池实现
    - **health.go**: 健康检查实现
    - **tls.go**: TLS安全连接实现
  - **middleware/**: 中间件功能
    - **circuit_breaker.go**: 断路器实现
    - **retry.go**: 重试机制实现
    - **metrics.go**: 性能指标收集
    - **rate_limiter.go**: 速率限制实现
  - **logger/**: 日志系统
    - **logger.go**: 结构化日志实现
- **proto/**: 协议缓冲区定义和生成的代码
- **main.go**: 版本信息

## 设计理念

客户端库采用模块化设计，各组件职责明确：

- **core**: 负责核心配置和客户端API
- **services**: 封装各种服务的API调用
- **conn**: 专注于连接管理和健康监控
- **middleware**: 提供横切关注点功能如重试、熔断等

这种架构使得各组件可以独立维护和测试，同时通过entry.go统一导出API，对外保持简洁的接口。

## 使用说明

这个库作为外部依赖导入到你的项目中使用，不提供命令行功能。所有功能通过编程方式使用，详见上述示例。

## 贡献

欢迎贡献代码和提出问题！请提交 Pull Request 或在 Issues 中反馈问题。

## 许可证

MIT 许可证 