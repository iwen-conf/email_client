# gRPC Email Client

一个功能齐全的 gRPC 邮件客户端库，为邮件服务和配置服务提供高级接口。

## 主要特性

- **统一连接管理**：使用单个连接同时访问邮件服务和配置服务
- **健康检查**：自动检测连接健康状态并进行自动重连
- **请求重试机制**：支持可配置的失败重试策略
- **断路器模式**：防止系统雪崩，自动中断连接到不健康的服务
- **性能指标收集**：监控请求执行情况和性能指标
- **选项模式配置**：灵活的客户端配置系统

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
// 使用 emails...

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

## 命令行工具

项目包含一个简单的命令行工具，可以用来测试连接和执行基本操作：

```bash
# 连接到指定服务器并启用调试日志
go run main.go --server=localhost:50051 --debug=true

# 启用健康检查和断路器
go run main.go --health=30s --circuit=true --failures=5 --reset=10s

# 列出所有配置
go run main.go --list-configs

# 列出所有发送的邮件
go run main.go --list-emails
```

## 高级功能说明

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

- **client/**: 核心客户端库
  - **client.go**: 主客户端实现
  - **email_service.go**: 邮件服务客户端
  - **config_service.go**: 配置服务客户端
  - **health.go**: 健康检查实现
  - **retry.go**: 重试机制实现
  - **circuit_breaker.go**: 断路器实现
  - **metrics.go**: 性能指标收集
  - **options.go**: 客户端选项系统
- **proto/**: 协议缓冲区定义和生成的代码
- **main.go**: 命令行工具实现

## 贡献

欢迎贡献代码和提出问题！请提交 Pull Request 或在 Issues 中反馈问题。

## 许可证

MIT 许可证 