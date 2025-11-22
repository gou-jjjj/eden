package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var DefaultLogger, _ = NewLogger(true, "./logs", "default_logger")

// Logger 日志接口
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
}

// LogLevel 日志级别
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// DocxLogger 日志记录器
type DocxLogger struct {
	needStdio bool
	filePath  string
	file      *os.File
	mu        sync.Mutex
	level     LogLevel
}

// NewLogger 创建新的日志记录器
func NewLogger(needStdio bool, outputDir, fileName string) (*DocxLogger, error) {
	// 确保输出目录存在
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 生成日志文件名：原文件名_翻译时间戳_log.txt
	logFileName := fmt.Sprintf("%s_log.txt", fileName)
	filePath := filepath.Join(outputDir, logFileName)

	// 创建日志文件
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("创建日志文件失败: %v", err)
	}

	return &DocxLogger{
		filePath:  filePath,
		file:      file,
		level:     INFO,
		needStdio: needStdio,
	}, nil
}

// SetLevel 设置日志级别
func (l *DocxLogger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// writeLog 写入日志的内部方法
func (l *DocxLogger) writeLog(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 格式化时间戳
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// 日志级别字符串
	levelStr := ""
	switch level {
	case DEBUG:
		levelStr = "DEBUG"
	case INFO:
		levelStr = "INFO"
	case WARN:
		levelStr = "WARN"
	case ERROR:
		levelStr = "ERROR"
	}

	// 格式化消息
	message := fmt.Sprintf(format, args...)
	logEntry := fmt.Sprintf("[%s] [%s] %s\n", timestamp, levelStr, message)

	// 写入文件
	if _, err := l.file.WriteString(logEntry); err != nil {
		fmt.Printf("写入日志失败: %v\n", err)
	}

	// 同时输出到控制台
	if l.needStdio {
		fmt.Print(logEntry)
	}
}

// Debug 记录调试信息
func (l *DocxLogger) Debug(format string, args ...interface{}) {
	l.writeLog(DEBUG, format, args...)
}

// Info 记录一般信息
func (l *DocxLogger) Info(format string, args ...interface{}) {
	l.writeLog(INFO, format, args...)
}

// Warn 记录警告信息
func (l *DocxLogger) Warn(format string, args ...interface{}) {
	l.writeLog(WARN, format, args...)
}

// Error 记录错误信息
func (l *DocxLogger) Error(format string, args ...interface{}) {
	l.writeLog(ERROR, format, args...)
}

// Close 关闭日志记录器
func (l *DocxLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		_ = l.file.Sync()
		return l.file.Close()
	}
	return nil
}

// GetLogFilePath 获取日志文件路径
func (l *DocxLogger) GetLogFilePath() string {
	return l.filePath
}
