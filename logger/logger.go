package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

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
	timestamp := time.Now().Format("20060102_150405")
	logFileName := fmt.Sprintf("%s_%s_log.txt", fileName, timestamp)
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

// LogTranslationStart 记录翻译开始
func (l *DocxLogger) LogTranslationStart(inputPath, fromLang, toLang string) {
	l.Info("=== 翻译任务开始 ===")
	l.Info("输入文件: %s", inputPath)
	l.Info("源语言: %s", fromLang)
	l.Info("目标语言: %s", toLang)
	l.Info("开始时间: %s", time.Now().Format("2006-01-02 15:04:05"))
}

// LogTranslationEnd 记录翻译结束
func (l *DocxLogger) LogTranslationEnd(outputPath string, success bool, duration time.Duration) {
	if success {
		l.Info("=== 翻译任务完成 ===")
		l.Info("输出文件: %s", outputPath)
		l.Info("耗时: %v", duration)
		l.Info("状态: 成功")
	} else {
		l.Error("=== 翻译任务失败 ===")
		l.Error("耗时: %v", duration)
		l.Error("状态: 失败")
	}
}

// LogFileLoad 记录文件加载
func (l *DocxLogger) LogFileLoad(success bool, filePath string, err error) {
	if success {
		l.Info("文件加载成功: %s", filePath)
	} else {
		l.Error("文件加载失败: %s, 错误: %v", filePath, err)
	}
}

// LogTextExtraction 记录文本提取
func (l *DocxLogger) LogTextExtraction(paragraphCount, tableCount int) {
	l.Info("文本提取完成")
	l.Info("段落数量: %d", paragraphCount)
	l.Info("表格数量: %d", tableCount)
}

// LogParagraphProcessing 记录段落处理
func (l *DocxLogger) LogParagraphProcessing(paragraphIndex int, originalText string, needTranslation bool) {
	if needTranslation {
		l.Debug("段落 %d [%s]", paragraphIndex, originalText)
	} else {
		l.Debug("段落 %d 跳过翻译: %s", paragraphIndex, originalText)
	}
}

// LogTranslationRequest 记录翻译请求
func (l *DocxLogger) LogTranslationRequest(paragraphIndex int, fromLang, toLang string, text string) {
	l.Info("发送翻译请求 - 段落 %d", paragraphIndex)
	l.Debug("翻译内容: %s", text)
}

// LogTranslationResponse 记录翻译响应
func (l *DocxLogger) LogTranslationResponse(paragraphIndex int, success bool, translatedText string, err error) {
	if success {
		l.Info("翻译完成 - 段落 %d", paragraphIndex)
		l.Debug("翻译结果: %s", translatedText)
	} else {
		l.Error("翻译失败 - 段落 %d, 错误: %v", paragraphIndex, err)
	}
}

// LogFileSave 记录文件保存
func (l *DocxLogger) LogFileSave(success bool, outputPath string, err error) {
	if success {
		l.Info("文件保存成功: %s", outputPath)
	} else {
		l.Error("文件保存失败: %s, 错误: %v", outputPath, err)
	}
}

// LogStatistics 记录统计信息
func (l *DocxLogger) LogStatistics(totalParagraphs, translatedParagraphs, skippedParagraphs int) {
	l.Info("=== 翻译统计 ===")
	l.Info("总段落数: %d", totalParagraphs)
	l.Info("已翻译: %d", translatedParagraphs)
	l.Info("跳过翻译: %d", skippedParagraphs)
	l.Info("翻译率: %.2f%%", float64(translatedParagraphs)/float64(totalParagraphs)*100)
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
