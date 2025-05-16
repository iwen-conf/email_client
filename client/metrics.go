package client

import (
	"sync"
	"sync/atomic"
	"time"
)

// ClientMetrics 定义客户端的性能指标收集功能
type ClientMetrics struct {
	// 基本请求统计
	RequestCount        int64 // 总请求数
	RequestSuccessCount int64 // 成功请求数
	RequestFailureCount int64 // 失败请求数

	// 时间统计
	TotalRequestTime time.Duration // 所有请求耗时总和
	MinRequestTime   time.Duration // 最小请求耗时
	MaxRequestTime   time.Duration // 最大请求耗时

	// 最近错误统计
	lastErrors    []error // 最近的错误
	maxLastErrors int     // 最多保存的错误数量

	mutex sync.RWMutex // 保护访问统计数据的互斥锁
}

// NewClientMetrics 创建一个新的 ClientMetrics 实例
func NewClientMetrics(maxLastErrors int) *ClientMetrics {
	if maxLastErrors <= 0 {
		maxLastErrors = 10 // 默认保存最近 10 个错误
	}

	return &ClientMetrics{
		lastErrors:     make([]error, 0, maxLastErrors),
		maxLastErrors:  maxLastErrors,
		MinRequestTime: time.Duration(1<<63 - 1), // 初始值设为最大值
	}
}

// RecordRequest 记录请求结果
func (m *ClientMetrics) RecordRequest(success bool, latency time.Duration) {
	// 使用原子操作更新计数器
	atomic.AddInt64(&m.RequestCount, 1)

	if success {
		atomic.AddInt64(&m.RequestSuccessCount, 1)
	} else {
		atomic.AddInt64(&m.RequestFailureCount, 1)
	}

	// 更新时间统计需要锁保护
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.TotalRequestTime += latency

	// 更新最小请求时间
	if latency < m.MinRequestTime {
		m.MinRequestTime = latency
	}

	// 更新最大请求时间
	if latency > m.MaxRequestTime {
		m.MaxRequestTime = latency
	}
}

// RecordError 记录错误
func (m *ClientMetrics) RecordError(err error) {
	if err == nil {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 保持最多 maxLastErrors 个错误记录
	if len(m.lastErrors) >= m.maxLastErrors {
		// 移除最旧的错误
		m.lastErrors = m.lastErrors[1:]
	}

	// 添加新错误
	m.lastErrors = append(m.lastErrors, err)
}

// GetStats 获取统计指标快照
func (m *ClientMetrics) GetStats() ClientMetricsSnapshot {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var avgRequestTime time.Duration
	requestCount := atomic.LoadInt64(&m.RequestCount)
	if requestCount > 0 {
		avgRequestTime = m.TotalRequestTime / time.Duration(requestCount)
	}

	return ClientMetricsSnapshot{
		RequestCount:        atomic.LoadInt64(&m.RequestCount),
		RequestSuccessCount: atomic.LoadInt64(&m.RequestSuccessCount),
		RequestFailureCount: atomic.LoadInt64(&m.RequestFailureCount),
		SuccessRate:         float64(atomic.LoadInt64(&m.RequestSuccessCount)) / float64(requestCount),
		AvgRequestTime:      avgRequestTime,
		MinRequestTime:      m.MinRequestTime,
		MaxRequestTime:      m.MaxRequestTime,
	}
}

// Reset 重置所有统计数据
func (m *ClientMetrics) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	atomic.StoreInt64(&m.RequestCount, 0)
	atomic.StoreInt64(&m.RequestSuccessCount, 0)
	atomic.StoreInt64(&m.RequestFailureCount, 0)

	m.TotalRequestTime = 0
	m.MinRequestTime = time.Duration(1<<63 - 1)
	m.MaxRequestTime = 0

	m.lastErrors = make([]error, 0, m.maxLastErrors)
}

// ClientMetricsSnapshot 提供指标的不可变快照，便于安全地读取统计数据
type ClientMetricsSnapshot struct {
	RequestCount        int64         // 总请求数
	RequestSuccessCount int64         // 成功请求数
	RequestFailureCount int64         // 失败请求数
	SuccessRate         float64       // 成功率 (0.0-1.0)
	AvgRequestTime      time.Duration // 平均请求时间
	MinRequestTime      time.Duration // 最小请求时间
	MaxRequestTime      time.Duration // 最大请求时间
}
