package conn

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

// Manager 管理gRPC连接的生命周期
type Manager struct {
	conn            *grpc.ClientConn
	target          string
	connectionMutex sync.Mutex
	healthChecker   *HealthChecker
	healthCheckLock sync.Mutex
	debug           bool
}

// ManagerOption 定义连接管理器配置选项
type ManagerOption func(*Manager)

// WithHealthCheck 配置健康检查
func WithHealthCheck(enabled bool, interval time.Duration) ManagerOption {
	return func(m *Manager) {
		if enabled && interval > 0 {
			m.startHealthCheck(interval)
		}
	}
}

// NewManager 创建新的连接管理器
func NewManager(target string, timeout time.Duration, debug bool, opts ...ManagerOption) (*Manager, error) {
	if target == "" {
		return nil, fmt.Errorf("gRPC 连接目标地址不能为空")
	}

	m := &Manager{
		target: target,
		debug:  debug,
	}

	// 建立连接
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := m.connect(ctx); err != nil {
		return nil, err
	}

	// 应用选项
	for _, opt := range opts {
		opt(m)
	}

	return m, nil
}

// connect 建立 gRPC 连接
func (m *Manager) connect(ctx context.Context) error {
	m.connectionMutex.Lock()
	defer m.connectionMutex.Unlock()

	// 使用最新的gRPC连接语法
	conn, err := grpc.NewClient(m.target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Printf("[ERROR] Manager.connect: 连接 gRPC 服务失败 (%s): %v", m.target, err)
		return fmt.Errorf("连接 gRPC 服务失败 (%s): %w", m.target, err)
	}

	// 主动连接并等待 Ready 状态
	conn.Connect()
	if m.debug {
		log.Printf("[INFO] Manager.connect: 正在等待连接变为 Ready 状态 (%s)...", m.target)
	}

	// 循环等待连接就绪
	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			if m.debug {
				log.Printf("[INFO] Manager.connect: 成功连接到 gRPC 服务 (%s)", m.target)
			}
			break // 成功连接
		}
		if !conn.WaitForStateChange(ctx, state) {
			conn.Close()
			errMsg := fmt.Sprintf("等待连接状态变化超时或被取消 (%s)", m.target)
			log.Printf("[ERROR] Manager.connect: %s", errMsg)
			return fmt.Errorf("%s", errMsg)
		}
		currentState := conn.GetState()
		if m.debug {
			log.Printf("[DEBUG] Manager.connect: 连接状态变化 (%s): %v -> %v", m.target, state, currentState)
		}
		if currentState == connectivity.TransientFailure || currentState == connectivity.Shutdown {
			conn.Close()
			errMsg := fmt.Sprintf("连接失败，当前状态: %v (%s)", currentState, m.target)
			log.Printf("[ERROR] Manager.connect: %s", errMsg)
			return fmt.Errorf("%s", errMsg)
		}
	}

	m.conn = conn
	return nil
}

// Reconnect 重新建立 gRPC 连接
func (m *Manager) Reconnect(ctx context.Context, target string) error {
	// 更新目标地址（如果提供了）
	if target != "" {
		m.target = target
	}

	// 关闭旧连接
	if m.conn != nil {
		m.conn.Close()
	}

	return m.connect(ctx)
}

// GetConn 获取gRPC连接
func (m *Manager) GetConn() *grpc.ClientConn {
	return m.conn
}

// GetState 获取连接状态
func (m *Manager) GetState() connectivity.State {
	if m.conn == nil {
		return connectivity.Shutdown
	}
	return m.conn.GetState()
}

// Close 关闭连接
func (m *Manager) Close() error {
	// 停止健康检查
	m.stopHealthCheck()

	m.connectionMutex.Lock()
	defer m.connectionMutex.Unlock()

	if m.conn != nil {
		if m.debug {
			log.Printf("[INFO] Manager.Close: 正在关闭 gRPC 连接: %s", m.conn.Target())
		}
		err := m.conn.Close()
		m.conn = nil
		return err
	}
	return nil
}

// startHealthCheck 启动健康检查
func (m *Manager) startHealthCheck(interval time.Duration) {
	m.healthCheckLock.Lock()
	defer m.healthCheckLock.Unlock()

	if m.healthChecker != nil {
		m.healthChecker.Stop()
	}

	m.healthChecker = NewHealthChecker(m, interval, m.debug)
	m.healthChecker.Start()
}

// stopHealthCheck 停止健康检查
func (m *Manager) stopHealthCheck() {
	m.healthCheckLock.Lock()
	defer m.healthCheckLock.Unlock()

	if m.healthChecker != nil {
		m.healthChecker.Stop()
		m.healthChecker = nil
	}
}
