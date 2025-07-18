package core

import (
	"errors"
)

// ErrEmptyGrpcAddress 定义常见错误
var (
	ErrEmptyGrpcAddress = errors.New("gRPC 服务地址不能为空")

	// ErrHealthServiceNotInitialized 表示健康检查服务未初始化
	ErrHealthServiceNotInitialized = errors.New("健康检查服务未初始化")
)
