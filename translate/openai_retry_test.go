package translate

import (
	"errors"
	"fmt"
	"math"
	"net"
	"testing"
	"time"

	"github.com/gou-jjjj/eden/lang"
	"github.com/gou-jjjj/eden/logger"
)

// MockLogger 模拟日志记录器
type MockLogger struct {
	debugLogs []string
	infoLogs  []string
	warnLogs  []string
	errorLogs []string
}

func (m *MockLogger) Debug(format string, args ...interface{}) {
	m.debugLogs = append(m.debugLogs, fmt.Sprintf(format, args...))
}

func (m *MockLogger) Info(format string, args ...interface{}) {
	m.infoLogs = append(m.infoLogs, fmt.Sprintf(format, args...))
}

func (m *MockLogger) Warn(format string, args ...interface{}) {
	m.warnLogs = append(m.warnLogs, fmt.Sprintf(format, args...))
}

func (m *MockLogger) Error(format string, args ...interface{}) {
	m.errorLogs = append(m.errorLogs, fmt.Sprintf(format, args...))
}

// MockTranslator 模拟翻译器，用于测试重试逻辑
type MockTranslator struct {
	attempts     int
	maxAttempts  int
	shouldFail   bool
	networkError bool
}

func (m *MockTranslator) T(req *TranReq) (Paragraph, error) {
	m.attempts++

	// 如果设置了shouldFail，直接返回错误
	if m.shouldFail {
		if m.networkError {
			return nil, &net.DNSError{Err: "network error", Name: "test.com", Server: "8.8.8.8"}
		}
		return nil, errors.New("translation failed")
	}

	// 如果还没有达到成功所需的尝试次数，返回临时错误
	if m.attempts < m.maxAttempts {
		return nil, &net.DNSError{Err: "temporary network error", Name: "test.com", Server: "8.8.8.8"}
	}

	// 达到成功所需的尝试次数，返回成功结果
	return Paragraph{"Hello", "World"}, nil
}

func (m *MockTranslator) Name() string {
	return "MockTranslator"
}

// MockTranOpenai 模拟TranOpenai，用于测试
type MockTranOpenai struct {
	*MockTranslator
	retryConfig RetryConfig
	logger      logger.Logger
}

func (m *MockTranOpenai) translateWithRetry(req *TranReq) (Paragraph, error) {
	var lastErr error

	for attempt := 0; attempt <= m.retryConfig.MaxRetries; attempt++ {
		// 如果不是第一次尝试，等待一段时间
		if attempt > 0 {
			delay := m.calculateDelay(attempt - 1)
			if m.logger != nil {
				m.logger.Debug("重试翻译，第 %d 次尝试，等待 %v", attempt, delay)
			}
			time.Sleep(delay)
		}

		// 记录翻译尝试
		if m.logger != nil {
			if attempt == 0 {
				m.logger.Debug("开始翻译请求")
			} else {
				m.logger.Debug("重试翻译，第 %d 次尝试", attempt)
			}
		}

		// 尝试翻译
		result, err := m.MockTranslator.T(req)
		if err == nil {
			if m.logger != nil {
				if attempt > 0 {
					m.logger.Info("翻译重试成功，第 %d 次尝试", attempt)
				} else {
					m.logger.Debug("翻译成功")
				}
			}
			return result, nil
		}

		lastErr = err

		// 记录错误
		if m.logger != nil {
			m.logger.Warn("翻译失败，第 %d 次尝试，错误: %v", attempt+1, err)
		}

		// 检查是否应该重试
		if !isRetryableError(err) {
			if m.logger != nil {
				m.logger.Warn("错误不可重试，停止重试: %v", err)
			}
			break
		}

		// 如果还有重试机会，继续
		if attempt < m.retryConfig.MaxRetries {
			if m.logger != nil {
				m.logger.Info("错误可重试，准备重试，剩余重试次数: %d", m.retryConfig.MaxRetries-attempt)
			}
			continue
		}
	}

	if m.logger != nil {
		m.logger.Error("翻译失败，已用尽所有重试次数，最后错误: %v", lastErr)
	}

	return nil, lastErr
}

func (m *MockTranOpenai) calculateDelay(attempt int) time.Duration {
	delay := float64(m.retryConfig.BaseDelay) * math.Pow(m.retryConfig.BackoffFactor, float64(attempt))
	if delay > float64(m.retryConfig.MaxDelay) {
		delay = float64(m.retryConfig.MaxDelay)
	}
	return time.Duration(delay)
}

func TestRetryConfig(t *testing.T) {
	config := RetryConfig{
		MaxRetries:    5,
		BaseDelay:     100 * time.Millisecond,
		MaxDelay:      2 * time.Second,
		BackoffFactor: 2.0,
	}

	translator := &TranOpenai{
		retryConfig: config,
	}

	// 测试延迟计算
	delay1 := translator.calculateDelay(0)
	if delay1 != 100*time.Millisecond {
		t.Errorf("Expected delay 100ms, got %v", delay1)
	}

	delay2 := translator.calculateDelay(1)
	if delay2 != 200*time.Millisecond {
		t.Errorf("Expected delay 200ms, got %v", delay2)
	}

	delay3 := translator.calculateDelay(2)
	if delay3 != 400*time.Millisecond {
		t.Errorf("Expected delay 400ms, got %v", delay3)
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Network timeout error",
			err:      &net.DNSError{Err: "timeout", Name: "test.com", Server: "8.8.8.8"},
			expected: true,
		},
		{
			name:     "Rate limit error",
			err:      errors.New("rate limit exceeded"),
			expected: true,
		},
		{
			name:     "Service unavailable error",
			err:      errors.New("service unavailable"),
			expected: true,
		},
		{
			name:     "Authentication error",
			err:      errors.New("authentication failed"),
			expected: false,
		},
		{
			name:     "Invalid request error",
			err:      errors.New("invalid request"),
			expected: false,
		},
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("isRetryableError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTranslateWithRetry_Success(t *testing.T) {
	logger := &MockLogger{}
	config := RetryConfig{
		MaxRetries:    3,
		BaseDelay:     10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	translator := &MockTranOpenai{
		MockTranslator: &MockTranslator{maxAttempts: 1, shouldFail: false},
		retryConfig:    config,
		logger:         logger,
	}

	req := &TranReq{
		From:  lang.EN,
		To:    lang.ZH,
		Paras: Paragraph{"Hello", "World"},
	}

	result, err := translator.translateWithRetry(req)
	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}

	if len(result) == 0 {
		t.Error("Expected non-empty result")
	}

	// 检查日志记录 - 第一次成功时可能只有debug日志
	if len(logger.debugLogs) == 0 {
		t.Error("Expected debug logs to be recorded")
	}

	// 验证重试次数
	if translator.MockTranslator.attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", translator.MockTranslator.attempts)
	}
}

func TestTranslateWithRetry_NonRetryableError(t *testing.T) {
	logger := &MockLogger{}
	config := RetryConfig{
		MaxRetries:    3,
		BaseDelay:     10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	translator := &MockTranOpenai{
		MockTranslator: &MockTranslator{shouldFail: true, networkError: false},
		retryConfig:    config,
		logger:         logger,
	}

	req := &TranReq{
		From:  lang.EN,
		To:    lang.ZH,
		Paras: Paragraph{"Hello", "World"},
	}

	result, err := translator.translateWithRetry(req)
	if err == nil {
		t.Error("Expected error, got success")
	}

	if result != nil {
		t.Error("Expected nil result")
	}

	// 检查是否记录了不可重试的错误
	warnFound := false
	for _, log := range logger.warnLogs {
		if contains(log, "错误不可重试") {
			warnFound = true
			break
		}
	}
	if !warnFound {
		t.Error("Expected warning about non-retryable error")
	}
}

func TestTranslateWithRetry_MaxRetriesExceeded(t *testing.T) {
	logger := &MockLogger{}
	config := RetryConfig{
		MaxRetries:    2,
		BaseDelay:     10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	translator := &MockTranOpenai{
		MockTranslator: &MockTranslator{maxAttempts: 10, shouldFail: true, networkError: true},
		retryConfig:    config,
		logger:         logger,
	}

	req := &TranReq{
		From:  lang.EN,
		To:    lang.ZH,
		Paras: Paragraph{"Hello", "World"},
	}

	result, err := translator.translateWithRetry(req)
	if err == nil {
		t.Error("Expected error, got success")
	}

	if result != nil {
		t.Error("Expected nil result")
	}

	// 检查是否记录了重试失败的错误
	errorFound := false
	for _, log := range logger.errorLogs {
		if contains(log, "已用尽所有重试次数") {
			errorFound = true
			break
		}
	}
	if !errorFound {
		t.Error("Expected error log about max retries exceeded")
	}
}

func TestNewOpenaiWithRetryAndLogger(t *testing.T) {
	logger := &MockLogger{}
	config := RetryConfig{
		MaxRetries:    2,
		BaseDelay:     10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	translator := NewOpenaiWithRetryAndLogger(OpenRouter, config, logger)
	if translator == nil {
		t.Fatal("Expected non-nil translator")
	}

	if translator.retryConfig.MaxRetries != config.MaxRetries {
		t.Errorf("Expected MaxRetries %d, got %d", config.MaxRetries, translator.retryConfig.MaxRetries)
	}

	if translator.logger != logger {
		t.Error("Expected logger to be set")
	}
}

func TestNewOpenaiWithLogger(t *testing.T) {
	logger := &MockLogger{}

	translator := NewOpenaiWithLogger(OpenRouter, logger)
	if translator == nil {
		t.Fatal("Expected non-nil translator")
	}

	if translator.logger != logger {
		t.Error("Expected logger to be set")
	}

	// 应该使用默认重试配置
	if translator.retryConfig.MaxRetries != DefaultRetryConfig.MaxRetries {
		t.Errorf("Expected default MaxRetries %d, got %d", DefaultRetryConfig.MaxRetries, translator.retryConfig.MaxRetries)
	}
}

// 辅助函数
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
