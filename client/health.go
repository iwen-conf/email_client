package client

import (
	"context"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc/connectivity"
)

// HealthChecker 提供连接健康检查和自动重连功能
type HealthChecker struct {
	client      *EmailClient
	stopChan    chan struct{}
	interval    time.Duration
	mutex       sync.RWMutex
	isRunning   bool
	onReconnect func() // 可选的重连回调
	debug       bool
}

// NewHealthChecker 创建新的健康检查器
func NewHealthChecker(client *EmailClient, interval time.Duration, debug bool) *HealthChecker {
	if interval <= 0 {
		interval = 30 * time.Second // 默认30秒检查一次
	}

	return &HealthChecker{
		client:   client,
		interval: interval,
		stopChan: make(chan struct{}),
		debug:    debug,
	}
}

// SetReconnectCallback 设置重连成功后的回调函数
func (h *HealthChecker) SetReconnectCallback(cb func()) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.onReconnect = cb
}

// Start 开始健康检查
func (h *HealthChecker) Start() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if h.isRunning {
		return
	}

	h.isRunning = true
	go h.run()

	if h.debug {
		log.Printf("[INFO] HealthChecker: 已启动健康检查，间隔: %v", h.interval)
	}
}

// Stop 停止健康检查
func (h *HealthChecker) Stop() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if !h.isRunning {
		return
	}

	close(h.stopChan)
	h.isRunning = false

	if h.debug {
		log.Printf("[INFO] HealthChecker: 健康检查已停止")
	}
}

// run 运行健康检查循环
func (h *HealthChecker) run() {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.checkHealth()
		case <-h.stopChan:
			return
		}
	}
}

// checkHealth 检查连接健康状态并在必要时进行重连
func (h *HealthChecker) checkHealth() {
	if h.client == nil || h.client.conn == nil {
		return
	}

	state := h.client.conn.GetState()
	if h.debug {
		log.Printf("[DEBUG] HealthChecker: 当前连接状态: %v", state)
	}

	// 当连接处于失败或关闭状态时进行重连
	if state == connectivity.TransientFailure || state == connectivity.Shutdown {
		if h.debug {
			log.Printf("[INFO] HealthChecker: 检测到连接异常状态: %v, 正在尝试重连", state)
		}
		h.reconnect()
	}
}

// reconnect 重新建立连接
func (h *HealthChecker) reconnect() {
	if h.client == nil {
		return
	}

	// 记录旧连接的目标地址，用于重连
	var target string
	if h.client.conn != nil {
		target = h.client.conn.Target()
	}

	if target == "" {
		if h.debug {
			log.Printf("[ERROR] HealthChecker: 无法重连，目标地址为空")
		}
		return
	}

	if h.debug {
		log.Printf("[INFO] HealthChecker: 正在尝试重连到 %s", target)
	}

	// 尝试重新连接
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// 连接已断开，先尝试关闭旧连接
	if h.client.conn != nil {
		h.client.conn.Close()
	}

	// 这里不能直接创建新的 EmailClient，因为我们需要保持已有的实例不变
	// 所以直接尝试重新建立 gRPC 连接
	// 注意：这里需要修改 EmailClient 结构，添加一个 reconnect 方法
	if err := h.client.reconnect(ctx, target); err != nil {
		if h.debug {
			log.Printf("[ERROR] HealthChecker: 重连失败: %v", err)
		}
		return
	}

	if h.debug {
		log.Printf("[INFO] HealthChecker: 重连成功")
	}

	// 调用重连回调
	h.mutex.RLock()
	onReconnect := h.onReconnect
	h.mutex.RUnlock()

	if onReconnect != nil {
		onReconnect()
	}
}
