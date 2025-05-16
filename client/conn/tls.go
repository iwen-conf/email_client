package conn

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// TLSConfig 定义TLS配置参数
type TLSConfig struct {
	// 是否启用TLS
	Enabled bool
	// 服务器主机名，用于证书验证
	ServerName string
	// 证书文件路径
	CertFile string
	// 密钥文件路径
	KeyFile string
	// CA证书文件路径
	CAFile string
	// 是否跳过TLS证书验证（仅用于开发/测试，生产环境不推荐）
	InsecureSkipVerify bool
}

// DefaultTLSConfig 返回默认TLS配置
func DefaultTLSConfig() TLSConfig {
	return TLSConfig{
		Enabled:            false,
		ServerName:         "",
		CertFile:           "",
		KeyFile:            "",
		CAFile:             "",
		InsecureSkipVerify: false,
	}
}

// CreateTLSCredentials 根据配置创建TLS凭据
func CreateTLSCredentials(config TLSConfig) (credentials.TransportCredentials, error) {
	if !config.Enabled {
		return nil, fmt.Errorf("TLS未启用")
	}

	var tlsConfig tls.Config

	// 设置服务器名称
	if config.ServerName != "" {
		tlsConfig.ServerName = config.ServerName
	}

	// 配置跳过证书验证选项
	tlsConfig.InsecureSkipVerify = config.InsecureSkipVerify

	// 加载CA证书
	if config.CAFile != "" {
		caBytes, err := os.ReadFile(config.CAFile)
		if err != nil {
			return nil, fmt.Errorf("读取CA证书文件失败: %w", err)
		}

		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(caBytes) {
			return nil, fmt.Errorf("添加CA证书到证书池失败")
		}
		tlsConfig.RootCAs = certPool
	}

	// 加载客户端证书和密钥（如果提供）
	if config.CertFile != "" && config.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("加载客户端证书和密钥失败: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return credentials.NewTLS(&tlsConfig), nil
}

// CreateTLSDialOption 创建TLS拨号选项
func CreateTLSDialOption(config TLSConfig) (grpc.DialOption, error) {
	if !config.Enabled {
		// 如果TLS未启用，返回不安全的凭据选项
		return grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true,
		})), nil
	}

	creds, err := CreateTLSCredentials(config)
	if err != nil {
		return nil, err
	}

	return grpc.WithTransportCredentials(creds), nil
}
