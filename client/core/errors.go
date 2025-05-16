package core

import (
	"errors"
)

// 定义常见错误
var (
	ErrEmptyGrpcAddress = errors.New("gRPC 服务地址不能为空")
)
