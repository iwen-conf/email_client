package conn

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

var (
	// ErrPoolClosed 表示连接池已关闭
	ErrPoolClosed = errors.New("连接池已关闭")
	// ErrConnectionTimeout 表示获取连接超时
	ErrConnectionTimeout = errors.New("获取连接超时")
)

// PoolConfig 定义连接池配置
type PoolConfig struct {
	// 初始连接数
	InitialSize int
	// 最大连接数
	MaxSize int
	// 最小空闲连接数
	MinIdle int
	// 连接最大空闲时间
	MaxIdle time.Duration
	// 获取连接超时时间
	AcquireTimeout time.Duration
	// 健康检查间隔
	HealthCheckInterval time.Duration
	// 是否启用健康检查
	EnableHealthCheck bool
	// 调试模式
	Debug bool
}

// DefaultPoolConfig 返回默认连接池配置
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		InitialSize:         2,
		MaxSize:             10,
		MinIdle:             1,
		MaxIdle:             5 * time.Minute,
		AcquireTimeout:      3 * time.Second,
		HealthCheckInterval: 30 * time.Second,
		EnableHealthCheck:   true,
		Debug:               false,
	}
}

// ConnectionPool 实现gRPC连接池
type ConnectionPool struct {
	// 连接池配置
	config PoolConfig
	// 目标地址
	target string
	// 连接池
	pool chan *PooledConnection
	// 连接计数
	counter int32
	// 锁
	mu sync.RWMutex
	// 连接池状态
	closed bool
	// 连接创建函数
	factory func() (*grpc.ClientConn, error)
	// 关闭时的回调函数
	onClose []func()
}

// PooledConnection 包装了连接及其元数据
type PooledConnection struct {
	*grpc.ClientConn
	pool      *ConnectionPool
	createdAt time.Time
	lastUsed  time.Time
	inUse     bool
}

// NewConnectionPool 创建一个新的连接池
func NewConnectionPool(target string, factory func() (*grpc.ClientConn, error), config PoolConfig) (*ConnectionPool, error) {
	if target == "" {
		return nil, fmt.Errorf("gRPC 目标地址不能为空")
	}

	if factory == nil {
		return nil, fmt.Errorf("连接工厂函数不能为空")
	}

	// 验证并调整配置
	if config.InitialSize < 0 {
		config.InitialSize = 0
	}
	if config.MaxSize < 1 {
		config.MaxSize = 1
	}
	if config.InitialSize > config.MaxSize {
		config.InitialSize = config.MaxSize
	}
	if config.MinIdle < 0 {
		config.MinIdle = 0
	}
	if config.MinIdle > config.MaxSize {
		config.MinIdle = config.MaxSize
	}

	p := &ConnectionPool{
		config:  config,
		target:  target,
		pool:    make(chan *PooledConnection, config.MaxSize),
		factory: factory,
		closed:  false,
	}

	// 初始化连接
	for i := 0; i < config.InitialSize; i++ {
		conn, err := p.createConnection()
		if err != nil {
			// 如果创建初始连接失败，关闭已创建的连接并返回错误
			p.Close()
			return nil, fmt.Errorf("初始化连接失败: %w", err)
		}
		p.pool <- conn
	}

	// 启动健康检查
	if config.EnableHealthCheck && config.HealthCheckInterval > 0 {
		go p.healthCheck()
	}

	if config.Debug {
		log.Printf("[INFO] ConnectionPool: 创建连接池成功，目标地址:%s, 初始连接数:%d, 最大连接数:%d",
			target, config.InitialSize, config.MaxSize)
	}

	return p, nil
}

// Get 从连接池获取一个连接
func (p *ConnectionPool) Get(ctx context.Context) (*PooledConnection, error) {
	if p.isClosed() {
		return nil, ErrPoolClosed
	}

	// 尝试从池中获取连接
	select {
	case conn, ok := <-p.pool:
		if !ok {
			return nil, ErrPoolClosed
		}
		// 检查连接状态
		if !p.isConnectionHealthy(conn) {
			if p.config.Debug {
				log.Printf("[DEBUG] ConnectionPool: 连接不健康，关闭并重新创建，目标地址:%s", p.target)
			}
			conn.ClientConn.Close()
			atomic.AddInt32(&p.counter, -1)
			return p.createOrAcquireConnection(ctx)
		}
		conn.inUse = true
		conn.lastUsed = time.Now()
		return conn, nil
	default:
		// 池中没有连接，检查是否可以创建新连接
		return p.createOrAcquireConnection(ctx)
	}
}

// createOrAcquireConnection 创建新连接或等待可用连接
func (p *ConnectionPool) createOrAcquireConnection(ctx context.Context) (*PooledConnection, error) {
	// 如果未达到最大连接数，创建新连接
	currentCount := atomic.LoadInt32(&p.counter)
	if currentCount < int32(p.config.MaxSize) {
		if atomic.CompareAndSwapInt32(&p.counter, currentCount, currentCount+1) {
			conn, err := p.createConnection()
			if err != nil {
				atomic.AddInt32(&p.counter, -1)
				return nil, err
			}
			conn.inUse = true
			conn.lastUsed = time.Now()
			return conn, nil
		}
	}

	// 尝试在超时内获取连接
	timeout := p.config.AcquireTimeout
	if deadline, ok := ctx.Deadline(); ok {
		ctxTimeout := time.Until(deadline)
		if ctxTimeout < timeout {
			timeout = ctxTimeout
		}
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case conn, ok := <-p.pool:
		if !ok {
			return nil, ErrPoolClosed
		}
		// 检查连接状态
		if !p.isConnectionHealthy(conn) {
			if p.config.Debug {
				log.Printf("[DEBUG] ConnectionPool: 连接不健康，关闭并重新创建，目标地址:%s", p.target)
			}
			conn.ClientConn.Close()
			atomic.AddInt32(&p.counter, -1)
			return p.createOrAcquireConnection(ctx)
		}
		conn.inUse = true
		conn.lastUsed = time.Now()
		return conn, nil
	case <-timer.C:
		return nil, ErrConnectionTimeout
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Put 将连接放回池中
func (p *ConnectionPool) Put(conn *PooledConnection) error {
	if conn == nil {
		return errors.New("连接不能为空")
	}

	if conn.pool != p {
		return errors.New("连接不属于此连接池")
	}

	if p.isClosed() {
		// 如果池已关闭，直接关闭连接
		conn.ClientConn.Close()
		atomic.AddInt32(&p.counter, -1)
		return nil
	}

	conn.inUse = false
	conn.lastUsed = time.Now()

	// 如果连接不健康，关闭并减少计数
	if !p.isConnectionHealthy(conn) {
		if p.config.Debug {
			log.Printf("[DEBUG] ConnectionPool: 放回不健康的连接，关闭并减少计数，目标地址:%s", p.target)
		}
		conn.ClientConn.Close()
		atomic.AddInt32(&p.counter, -1)
		return nil
	}

	// 尝试放回池中
	select {
	case p.pool <- conn:
		return nil
	default:
		// 池已满，关闭多余的连接
		if p.config.Debug {
			log.Printf("[DEBUG] ConnectionPool: 连接池已满，关闭多余连接，目标地址:%s", p.target)
		}
		conn.ClientConn.Close()
		atomic.AddInt32(&p.counter, -1)
		return nil
	}
}

// Close 关闭连接池中的所有连接
func (p *ConnectionPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	p.closed = true
	close(p.pool)

	// 关闭所有连接
	for conn := range p.pool {
		conn.ClientConn.Close()
		atomic.AddInt32(&p.counter, -1)
	}

	// 执行关闭回调
	for _, callback := range p.onClose {
		callback()
	}

	if p.config.Debug {
		log.Printf("[INFO] ConnectionPool: 连接池已关闭，目标地址:%s", p.target)
	}
}

// AddCloseCallback 添加连接池关闭时的回调函数
func (p *ConnectionPool) AddCloseCallback(callback func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onClose = append(p.onClose, callback)
}

// Stats 返回连接池统计信息
func (p *ConnectionPool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return PoolStats{
		TotalConnections: int(atomic.LoadInt32(&p.counter)),
		IdleConnections:  len(p.pool),
		MaxConnections:   p.config.MaxSize,
	}
}

// PoolStats 连接池统计信息
type PoolStats struct {
	TotalConnections int // 总连接数
	IdleConnections  int // 空闲连接数
	MaxConnections   int // 最大连接数
}

// createConnection 创建新连接
func (p *ConnectionPool) createConnection() (*PooledConnection, error) {
	conn, err := p.factory()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &PooledConnection{
		ClientConn: conn,
		pool:       p,
		createdAt:  now,
		lastUsed:   now,
		inUse:      false,
	}, nil
}

// healthCheck 定期检查连接池中的连接
func (p *ConnectionPool) healthCheck() {
	ticker := time.NewTicker(p.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		<-ticker.C

		if p.isClosed() {
			return
		}

		p.performHealthCheck()
	}
}

// performHealthCheck 执行一次健康检查
func (p *ConnectionPool) performHealthCheck() {
	if p.config.Debug {
		log.Printf("[DEBUG] ConnectionPool: 开始执行连接池健康检查，目标地址:%s", p.target)
	}

	// 统计需要保留的连接数量
	idleCount := len(p.pool)
	minIdle := p.config.MinIdle

	// 遍历所有空闲连接
	var tempConns []*PooledConnection
	idleTimeout := time.Now().Add(-p.config.MaxIdle)

	// 先将所有连接取出来
	drainCount := 0
	for drainCount < idleCount {
		select {
		case conn := <-p.pool:
			tempConns = append(tempConns, conn)
			drainCount++
		default:
			// 没有更多连接
			break
		}
	}

	// 检查连接，决定保留哪些
	var keepCount int
	for _, conn := range tempConns {
		// 检查连接是否过期或不健康
		if conn.lastUsed.Before(idleTimeout) || !p.isConnectionHealthy(conn) {
			if p.config.Debug {
				log.Printf("[DEBUG] ConnectionPool: 关闭过期或不健康的连接，目标地址:%s, 空闲时间:%v",
					p.target, time.Since(conn.lastUsed))
			}
			conn.ClientConn.Close()
			atomic.AddInt32(&p.counter, -1)
		} else {
			// 保留连接，放回池中
			if keepCount < p.config.MaxSize {
				select {
				case p.pool <- conn:
					keepCount++
				default:
					// 池已满，关闭多余连接
					conn.ClientConn.Close()
					atomic.AddInt32(&p.counter, -1)
				}
			} else {
				// 超过最大数量，关闭多余连接
				conn.ClientConn.Close()
				atomic.AddInt32(&p.counter, -1)
			}
		}
	}

	// 如果空闲连接数量少于最小值，创建新连接
	if keepCount < minIdle {
		needed := minIdle - keepCount
		for i := 0; i < needed; i++ {
			// 检查连接总数是否已达到上限
			if atomic.LoadInt32(&p.counter) >= int32(p.config.MaxSize) {
				break
			}

			atomic.AddInt32(&p.counter, 1)
			conn, err := p.createConnection()
			if err != nil {
				atomic.AddInt32(&p.counter, -1)
				if p.config.Debug {
					log.Printf("[ERROR] ConnectionPool: 健康检查期间创建新连接失败: %v", err)
				}
				continue
			}

			select {
			case p.pool <- conn:
				// 成功添加到池中
			default:
				// 池已满，关闭连接
				conn.ClientConn.Close()
				atomic.AddInt32(&p.counter, -1)
			}
		}
	}

	if p.config.Debug {
		log.Printf("[DEBUG] ConnectionPool: 连接池健康检查完成，目标地址:%s, 当前空闲连接:%d, 总连接:%d",
			p.target, len(p.pool), atomic.LoadInt32(&p.counter))
	}
}

// isConnectionHealthy 检查连接是否健康
func (p *ConnectionPool) isConnectionHealthy(conn *PooledConnection) bool {
	if conn == nil || conn.ClientConn == nil {
		return false
	}

	state := conn.ClientConn.GetState()
	return state != connectivity.TransientFailure && state != connectivity.Shutdown
}

// isClosed 检查连接池是否已关闭
func (p *ConnectionPool) isClosed() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.closed
}

// Release 释放连接，当连接不再需要或出现错误时调用
func (c *PooledConnection) Release() {
	if c != nil && c.pool != nil {
		c.pool.Put(c)
	}
}

// Age 返回连接已存在的时间
func (c *PooledConnection) Age() time.Duration {
	return time.Since(c.createdAt)
}

// IdleTime 返回连接空闲的时间
func (c *PooledConnection) IdleTime() time.Duration {
	if c.inUse {
		return 0
	}
	return time.Since(c.lastUsed)
}
