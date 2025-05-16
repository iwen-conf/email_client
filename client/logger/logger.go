package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"
)

// LogLevel 表示日志级别
type LogLevel int

const (
	// DebugLevel 用于详细的调试信息
	DebugLevel LogLevel = iota
	// InfoLevel 用于一般性信息
	InfoLevel
	// WarnLevel 用于警告信息
	WarnLevel
	// ErrorLevel 用于错误信息
	ErrorLevel
	// FatalLevel 用于致命错误信息
	FatalLevel
)

// String 返回日志级别的字符串表示
func (l LogLevel) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", int(l))
	}
}

// Color 返回日志级别的ANSI颜色代码
func (l LogLevel) Color() string {
	switch l {
	case DebugLevel:
		return "\033[36m" // Cyan
	case InfoLevel:
		return "\033[32m" // Green
	case WarnLevel:
		return "\033[33m" // Yellow
	case ErrorLevel:
		return "\033[31m" // Red
	case FatalLevel:
		return "\033[35m" // Magenta
	default:
		return "\033[37m" // White
	}
}

// Logger 定义日志记录器接口
type Logger interface {
	// Debug 记录调试级别的日志
	Debug(args ...interface{})
	// Debugf 记录格式化的调试级别日志
	Debugf(format string, args ...interface{})
	// Info 记录信息级别的日志
	Info(args ...interface{})
	// Infof 记录格式化的信息级别日志
	Infof(format string, args ...interface{})
	// Warn 记录警告级别的日志
	Warn(args ...interface{})
	// Warnf 记录格式化的警告级别日志
	Warnf(format string, args ...interface{})
	// Error 记录错误级别的日志
	Error(args ...interface{})
	// Errorf 记录格式化的错误级别日志
	Errorf(format string, args ...interface{})
	// Fatal 记录致命错误级别的日志
	Fatal(args ...interface{})
	// Fatalf 记录格式化的致命错误级别日志
	Fatalf(format string, args ...interface{})

	// WithField 添加单个字段并返回新的Logger
	WithField(key string, value interface{}) Logger
	// WithFields 添加多个字段并返回新的Logger
	WithFields(fields map[string]interface{}) Logger
	// WithRequestID 添加请求ID字段并返回新的Logger
	WithRequestID(requestID string) Logger
	// WithError 添加错误字段并返回新的Logger
	WithError(err error) Logger

	// SetLevel 设置日志级别
	SetLevel(level LogLevel)
	// SetOutput 设置日志输出
	SetOutput(out io.Writer)
	// SetFormatter 设置日志格式化器
	SetFormatter(formatter Formatter)
}

// Formatter 定义日志格式化器接口
type Formatter interface {
	Format(entry *Entry) ([]byte, error)
}

// Entry 表示单条日志记录
type Entry struct {
	Logger    *StandardLogger
	Level     LogLevel
	Time      time.Time
	Message   string
	Fields    map[string]interface{}
	RequestID string
}

// StandardLogger 是Logger接口的标准实现
type StandardLogger struct {
	mu        sync.Mutex
	out       io.Writer
	level     LogLevel
	formatter Formatter
	fields    map[string]interface{}
	requestID string
}

// TextFormatter 是一个简单的文本格式化器
type TextFormatter struct {
	// DisableColors 禁用颜色输出
	DisableColors bool
	// DisableTimestamp 禁用时间戳
	DisableTimestamp bool
	// TimeFormat 时间格式字符串
	TimeFormat string
	// EnableFieldAlignment 启用字段对齐
	EnableFieldAlignment bool
}

// JSONFormatter 将日志格式化为JSON
type JSONFormatter struct {
	// DisableTimestamp 禁用时间戳
	DisableTimestamp bool
	// TimeFormat 时间格式字符串
	TimeFormat string
}

// NewStandardLogger 创建一个新的标准日志记录器
func NewStandardLogger() *StandardLogger {
	return &StandardLogger{
		out:       os.Stderr,
		level:     InfoLevel,
		formatter: &TextFormatter{TimeFormat: time.RFC3339},
		fields:    make(map[string]interface{}),
	}
}

// SetLevel 设置日志级别
func (l *StandardLogger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetOutput 设置日志输出
func (l *StandardLogger) SetOutput(out io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.out = out
}

// SetFormatter 设置日志格式化器
func (l *StandardLogger) SetFormatter(formatter Formatter) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.formatter = formatter
}

// Debug 记录调试级别的日志
func (l *StandardLogger) Debug(args ...interface{}) {
	l.log(DebugLevel, fmt.Sprint(args...))
}

// Debugf 记录格式化的调试级别日志
func (l *StandardLogger) Debugf(format string, args ...interface{}) {
	l.log(DebugLevel, fmt.Sprintf(format, args...))
}

// Info 记录信息级别的日志
func (l *StandardLogger) Info(args ...interface{}) {
	l.log(InfoLevel, fmt.Sprint(args...))
}

// Infof 记录格式化的信息级别日志
func (l *StandardLogger) Infof(format string, args ...interface{}) {
	l.log(InfoLevel, fmt.Sprintf(format, args...))
}

// Warn 记录警告级别的日志
func (l *StandardLogger) Warn(args ...interface{}) {
	l.log(WarnLevel, fmt.Sprint(args...))
}

// Warnf 记录格式化的警告级别日志
func (l *StandardLogger) Warnf(format string, args ...interface{}) {
	l.log(WarnLevel, fmt.Sprintf(format, args...))
}

// Error 记录错误级别的日志
func (l *StandardLogger) Error(args ...interface{}) {
	l.log(ErrorLevel, fmt.Sprint(args...))
}

// Errorf 记录格式化的错误级别日志
func (l *StandardLogger) Errorf(format string, args ...interface{}) {
	l.log(ErrorLevel, fmt.Sprintf(format, args...))
}

// Fatal 记录致命错误级别的日志
func (l *StandardLogger) Fatal(args ...interface{}) {
	l.log(FatalLevel, fmt.Sprint(args...))
	os.Exit(1)
}

// Fatalf 记录格式化的致命错误级别日志
func (l *StandardLogger) Fatalf(format string, args ...interface{}) {
	l.log(FatalLevel, fmt.Sprintf(format, args...))
	os.Exit(1)
}

// WithField 添加单个字段并返回新的Logger
func (l *StandardLogger) WithField(key string, value interface{}) Logger {
	return l.WithFields(map[string]interface{}{key: value})
}

// WithFields 添加多个字段并返回新的Logger
func (l *StandardLogger) WithFields(fields map[string]interface{}) Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &StandardLogger{
		out:       l.out,
		level:     l.level,
		formatter: l.formatter,
		fields:    make(map[string]interface{}, len(l.fields)+len(fields)),
		requestID: l.requestID,
	}

	// 复制现有字段
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// 添加新字段
	for k, v := range fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

// WithRequestID 添加请求ID字段并返回新的Logger
func (l *StandardLogger) WithRequestID(requestID string) Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &StandardLogger{
		out:       l.out,
		level:     l.level,
		formatter: l.formatter,
		fields:    make(map[string]interface{}, len(l.fields)),
		requestID: requestID,
	}

	// 复制现有字段
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

// WithError 添加错误字段并返回新的Logger
func (l *StandardLogger) WithError(err error) Logger {
	if err == nil {
		return l
	}
	return l.WithField("error", err.Error())
}

// log 记录日志的内部方法
func (l *StandardLogger) log(level LogLevel, message string) {
	// 检查日志级别
	if level < l.level {
		return
	}

	// 创建日志条目
	entry := &Entry{
		Logger:    l,
		Level:     level,
		Time:      time.Now(),
		Message:   message,
		Fields:    l.fields,
		RequestID: l.requestID,
	}

	// 格式化日志条目
	data, err := l.formatter.Format(entry)
	if err != nil {
		// 格式化失败，尝试回退到简单格式
		fallbackData := []byte(fmt.Sprintf("[ERROR] 日志格式化失败: %v - 原始消息: %s\n", err, message))
		l.mu.Lock()
		defer l.mu.Unlock()
		l.out.Write(fallbackData)
		return
	}

	// 写入日志
	l.mu.Lock()
	defer l.mu.Unlock()
	l.out.Write(data)
}

// Format 格式化日志条目为文本格式
func (f *TextFormatter) Format(entry *Entry) ([]byte, error) {
	var sb strings.Builder

	// 时间戳
	if !f.DisableTimestamp {
		timeFormat := f.TimeFormat
		if timeFormat == "" {
			timeFormat = time.RFC3339
		}
		sb.WriteString(entry.Time.Format(timeFormat))
		sb.WriteString(" ")
	}

	// 日志级别，带颜色
	if !f.DisableColors {
		sb.WriteString(entry.Level.Color())
	}
	sb.WriteString("[")
	sb.WriteString(entry.Level.String())
	sb.WriteString("]")
	if !f.DisableColors {
		sb.WriteString("\033[0m") // Reset color
	}
	sb.WriteString(" ")

	// 请求ID
	if entry.RequestID != "" {
		sb.WriteString("[")
		sb.WriteString(entry.RequestID)
		sb.WriteString("] ")
	}

	// 消息
	sb.WriteString(entry.Message)

	// 字段
	if len(entry.Fields) > 0 {
		sb.WriteString(" ")
		var fieldStrs []string
		maxKeyLen := 0

		// 如果需要字段对齐，先计算最大键长度
		if f.EnableFieldAlignment {
			for k := range entry.Fields {
				if len(k) > maxKeyLen {
					maxKeyLen = len(k)
				}
			}
		}

		for k, v := range entry.Fields {
			var fieldStr string
			if f.EnableFieldAlignment {
				formatStr := fmt.Sprintf("%%-%ds=%%v", maxKeyLen)
				fieldStr = fmt.Sprintf(formatStr, k, v)
			} else {
				fieldStr = fmt.Sprintf("%s=%v", k, v)
			}
			fieldStrs = append(fieldStrs, fieldStr)
		}
		sb.WriteString(strings.Join(fieldStrs, " "))
	}

	sb.WriteString("\n")
	return []byte(sb.String()), nil
}

// Format 格式化日志条目为JSON格式
func (f *JSONFormatter) Format(entry *Entry) ([]byte, error) {
	data := make(map[string]interface{})

	// 时间戳
	if !f.DisableTimestamp {
		timeFormat := f.TimeFormat
		if timeFormat == "" {
			timeFormat = time.RFC3339
		}
		data["time"] = entry.Time.Format(timeFormat)
	}

	// 日志级别
	data["level"] = entry.Level.String()

	// 请求ID
	if entry.RequestID != "" {
		data["request_id"] = entry.RequestID
	}

	// 消息
	data["message"] = entry.Message

	// 字段
	for k, v := range entry.Fields {
		data[k] = v
	}

	// 转换为JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return append(jsonData, '\n'), nil
}

// 单例日志记录器，可以在包级别使用
var (
	std = NewStandardLogger()
)

// GetStandardLogger 返回标准日志记录器
func GetStandardLogger() *StandardLogger {
	return std
}

// SetLevel 设置默认日志记录器的级别
func SetLevel(level LogLevel) {
	std.SetLevel(level)
}

// SetOutput 设置默认日志记录器的输出
func SetOutput(out io.Writer) {
	std.SetOutput(out)
}

// SetFormatter 设置默认日志记录器的格式化器
func SetFormatter(formatter Formatter) {
	std.SetFormatter(formatter)
}

// Debug 使用默认日志记录器记录调试级别的日志
func Debug(args ...interface{}) {
	std.Debug(args...)
}

// Debugf 使用默认日志记录器记录格式化的调试级别日志
func Debugf(format string, args ...interface{}) {
	std.Debugf(format, args...)
}

// Info 使用默认日志记录器记录信息级别的日志
func Info(args ...interface{}) {
	std.Info(args...)
}

// Infof 使用默认日志记录器记录格式化的信息级别日志
func Infof(format string, args ...interface{}) {
	std.Infof(format, args...)
}

// Warn 使用默认日志记录器记录警告级别的日志
func Warn(args ...interface{}) {
	std.Warn(args...)
}

// Warnf 使用默认日志记录器记录格式化的警告级别日志
func Warnf(format string, args ...interface{}) {
	std.Warnf(format, args...)
}

// Error 使用默认日志记录器记录错误级别的日志
func Error(args ...interface{}) {
	std.Error(args...)
}

// Errorf 使用默认日志记录器记录格式化的错误级别日志
func Errorf(format string, args ...interface{}) {
	std.Errorf(format, args...)
}

// Fatal 使用默认日志记录器记录致命错误级别的日志
func Fatal(args ...interface{}) {
	std.Fatal(args...)
}

// Fatalf 使用默认日志记录器记录格式化的致命错误级别日志
func Fatalf(format string, args ...interface{}) {
	std.Fatalf(format, args...)
}

// WithField 使用默认日志记录器添加单个字段并返回新的Logger
func WithField(key string, value interface{}) Logger {
	return std.WithField(key, value)
}

// WithFields 使用默认日志记录器添加多个字段并返回新的Logger
func WithFields(fields map[string]interface{}) Logger {
	return std.WithFields(fields)
}

// WithRequestID 使用默认日志记录器添加请求ID字段并返回新的Logger
func WithRequestID(requestID string) Logger {
	return std.WithRequestID(requestID)
}

// WithError 使用默认日志记录器添加错误字段并返回新的Logger
func WithError(err error) Logger {
	return std.WithError(err)
}

// GenerateRequestID 生成唯一的请求ID
func GenerateRequestID() string {
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), randomString(8))
}

// 用于生成随机字符串
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
