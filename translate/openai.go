package translate

import (
	"context"
	"fmt"
	"math"
	"net"
	"os"
	"strings"
	"time"

	"github.com/gou-jjjj/eden/lang"
	"github.com/gou-jjjj/eden/logger"
	"github.com/gou-jjjj/eden/prompt"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

const (
	ZhiPu      = "zhipu"
	GithubFree = "githubfree"
	OpenRouter = "openrouter"
	AliBaBa    = "alibaba"

	Seq = "\n---\n"
)

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries    int           // 最大重试次数
	BaseDelay     time.Duration // 基础延迟时间
	MaxDelay      time.Duration // 最大延迟时间
	BackoffFactor float64       // 退避因子
}

// DefaultRetryConfig 默认重试配置
var DefaultRetryConfig = RetryConfig{
	MaxRetries:    3,
	BaseDelay:     time.Second,
	MaxDelay:      30 * time.Second,
	BackoffFactor: 2.0,
}

var OpenaiModelList = map[string]struct {
	Url   string
	Key   string
	Model string
}{
	ZhiPu:      {"https://open.bigmodel.cn/api/paas/v4", "23c650b6b73d4b1b80500699edcbf87c.qU9lcBTeGfku2gKz", "glm-4-plus"},
	GithubFree: {"https://api.chatanywhere.tech", "sk-vINYqBzbzrhdsFxZCO7MSSEvHL8tPradBhl77tLmWmEoTXs5", "deepseek-v3"},
	OpenRouter: {"https://openrouter.ai/api/v1", "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", "x-ai/grok-4-fast:free"},
	AliBaBa:    {"https://dashscope.aliyuncs.com/compatible-mode/v1", "sk-227cf58d893d4a689e82d2b8eb8f3564", "qwen-plus"},
}

type TranOpenai struct {
	url         string
	key         string
	model       string
	back        *TranOpenai
	retryConfig RetryConfig
	logger      logger.Logger
}

func NewOpenai(llmSource string, backTranOpenai ...*TranOpenai) *TranOpenai {
	s, ok := OpenaiModelList[llmSource]
	if !ok {
		return nil
	}

	return &TranOpenai{
		url:         s.Url,
		key:         s.Key,
		model:       s.Model,
		retryConfig: DefaultRetryConfig,
		back: func() *TranOpenai {
			if len(backTranOpenai) > 0 {
				return backTranOpenai[0]
			}
			return nil
		}(),
	}
}

// NewOpenaiWithRetry 创建带自定义重试配置的OpenAI翻译器
func NewOpenaiWithRetry(llmSource string, retryConfig RetryConfig, backTranOpenai ...*TranOpenai) *TranOpenai {
	s, ok := OpenaiModelList[llmSource]
	if !ok {
		return nil
	}

	return &TranOpenai{
		url:         s.Url,
		key:         s.Key,
		model:       s.Model,
		retryConfig: retryConfig,
		back: func() *TranOpenai {
			if len(backTranOpenai) > 0 {
				return backTranOpenai[0]
			}
			return nil
		}(),
	}
}

// NewOpenaiWithLogger 创建带日志记录器的OpenAI翻译器
func NewOpenaiWithLogger(llmSource string, logger logger.Logger, backTranOpenai ...*TranOpenai) *TranOpenai {
	s, ok := OpenaiModelList[llmSource]
	if !ok {
		return nil
	}

	return &TranOpenai{
		url:         s.Url,
		key:         s.Key,
		model:       s.Model,
		retryConfig: DefaultRetryConfig,
		logger:      logger,
		back: func() *TranOpenai {
			if len(backTranOpenai) > 0 {
				return backTranOpenai[0]
			}
			return nil
		}(),
	}
}

// NewOpenaiWithRetryAndLogger 创建带自定义重试配置和日志记录器的OpenAI翻译器
func NewOpenaiWithRetryAndLogger(llmSource string, retryConfig RetryConfig, logger logger.Logger, backTranOpenai ...*TranOpenai) *TranOpenai {
	s, ok := OpenaiModelList[llmSource]
	if !ok {
		return nil
	}

	return &TranOpenai{
		url:         s.Url,
		key:         s.Key,
		model:       s.Model,
		retryConfig: retryConfig,
		logger:      logger,
		back: func() *TranOpenai {
			if len(backTranOpenai) > 0 {
				return backTranOpenai[0]
			}
			return nil
		}(),
	}
}

// isRetryableError 判断错误是否可重试
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// 网络相关错误
	if netErr, ok := err.(net.Error); ok {
		return netErr.Temporary() || netErr.Timeout()
	}

	// 检查错误字符串中的常见可重试错误
	errStr := strings.ToLower(err.Error())
	retryablePatterns := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"network is unreachable",
		"temporary failure",
		"rate limit",
		"too many requests",
		"service unavailable",
		"internal server error",
		"bad gateway",
		"gateway timeout",
		"temporary",
		"retry",
		"response error",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// calculateDelay 计算重试延迟时间（指数退避）
func (t *TranOpenai) calculateDelay(attempt int) time.Duration {
	delay := float64(t.retryConfig.BaseDelay) * math.Pow(t.retryConfig.BackoffFactor, float64(attempt))
	if delay > float64(t.retryConfig.MaxDelay) {
		delay = float64(t.retryConfig.MaxDelay)
	}
	return time.Duration(delay)
}

// translateWithRetry 带重试的翻译方法
func (t *TranOpenai) translateWithRetry(req *TranReq) (Paragraph, error) {
	var lastErr error

	for attempt := 0; attempt <= t.retryConfig.MaxRetries; attempt++ {
		// 如果不是第一次尝试，等待一段时间
		if attempt > 0 {
			delay := t.calculateDelay(attempt - 1)
			if t.logger != nil {
				t.logger.Info("重试翻译，第 %d 次尝试，等待 %v", attempt, delay)
			}
			time.Sleep(delay)
		}

		// 记录翻译尝试
		if t.logger != nil {
			if attempt == 0 {
				t.logger.Debug("开始翻译请求")
			} else {
				t.logger.Info("重试翻译，第 %d 次尝试", attempt)
			}
		}

		// 尝试翻译
		result, err := t.performTranslation(req)
		if err == nil {
			if t.logger != nil {
				t.logger.Debug("翻译成功")
			}
			return result, nil
		}

		lastErr = err

		// 记录错误
		if t.logger != nil {
			t.logger.Warn("翻译失败，第 %d 次尝试，错误: %v", attempt+1, err)
		}

		// 检查是否应该重试
		if !isRetryableError(err) {
			if t.logger != nil {
				t.logger.Warn("错误不可重试，停止重试: %v", err)
			}
			break
		}

		// 如果还有重试机会，继续
		if attempt < t.retryConfig.MaxRetries {
			if t.logger != nil {
				t.logger.Info("错误可重试，准备重试，剩余重试次数: %d", t.retryConfig.MaxRetries-attempt)
			}
			continue
		}
	}

	// 如果所有重试都失败了，尝试备用翻译器
	if t.back != nil {
		if t.logger != nil {
			t.logger.Info("所有重试失败，尝试备用翻译器")
		}
		return t.back.T(req)
	}

	if t.logger != nil {
		t.logger.Error("翻译失败，已用尽所有重试次数，最后错误: %v", lastErr)
	}

	return nil, lastErr
}

// performTranslation 执行实际的翻译操作
func (t *TranOpenai) performTranslation(req *TranReq) (Paragraph, error) {
	ctx := context.Background()
	llm, err := openai.New(
		openai.WithBaseURL(t.url),
		openai.WithModel(t.model),
		openai.WithToken(t.key),
		openai.WithAPIType(openai.APITypeOpenAI),
	)
	if err != nil {
		return nil, err
	}

	msgs := samplePrompt[getLangKey(lang.ZH, lang.EN)]
	contentMsg := strings.Join(req.Paras, Seq)
	content := append([]llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, prompt.TranslatePrompt(req.From, req.To, len(req.Paras)))}, msgs...)
	content = append(content, llms.TextParts(llms.ChatMessageTypeHuman, contentMsg))
	generateContent, err := llm.GenerateContent(ctx, content)
	if err != nil {
		return nil, err
	}

	if len(generateContent.Choices) == 0 {
		return nil, fmt.Errorf("no response choices returned from API")
	}

	res := strings.Split(generateContent.Choices[0].Content, Seq)

	if len(req.Paras) != len(res) {
		if t.logger != nil {
			// 记录错误内容到日志文件，便于排查
			s := strings.Builder{}
			for i := 0; i < max(len(req.Paras), len(res)); i++ {
				if i < len(req.Paras) {
					s.WriteString(fmt.Sprintf("[%s] ", req.Paras[i]))
				} else {
					s.WriteString(fmt.Sprintf("[] "))
				}
				if i < len(res) {
					s.WriteString(fmt.Sprintf("[%s] ", res[i]))
				} else {
					s.WriteString("[] ")
				}
				s.WriteString(Seq)
			}

			_ = os.WriteFile(fmt.Sprintf("error_resp_%d.log", time.Now().Unix()), []byte(s.String()), 0644)
			t.logger.Warn("翻译结果段落数与请求段落数不匹配，可能存在部分翻译丢失，req:%d, res:%d", len(req.Paras), len(res))
		}
		return res, fmt.Errorf("response error，req:%d!=res:%d", len(req.Paras), len(res))
	}

	return res, nil
}

func (t *TranOpenai) T(req *TranReq) (Paragraph, error) {
	return t.translateWithRetry(req)
}

func (t *TranOpenai) Name() string {
	return "OpenAI"
}

func getLangKey(form, to string) string {
	return fmt.Sprintf("%s_%s", form, to)
}

var samplePrompt = map[string][]llms.MessageContent{
	getLangKey(lang.ZH, lang.EN): {
		llms.TextParts(llms.ChatMessageTypeHuman, "要运行程序，请使用：`python main.py --input data.json`\n---\n这将处理数据集并生成\n---\n输出文件到`/results/`目录，截止东部时间下午5点。"),
		llms.TextParts(llms.ChatMessageTypeAI, "To run the program, use: `python main.py --input data.json`\n---\nThis will process the dataset and generate\n---\noutput files in `/results/` directory by 5PM EST."),
	},
	getLangKey(lang.EN, lang.ZH): {
		llms.TextParts(llms.ChatMessageTypeHuman, "To run the program, use: `python main.py --input data.json`\n---\nThis will process the dataset and generate\n---\noutput files in `/results/` directory by 5PM EST."),
		llms.TextParts(llms.ChatMessageTypeAI, "要运行程序，请使用：`python main.py --input data.json`\n---\n这将处理数据集并生成\n---\n输出文件到`/results/`目录，截止东部时间下午5点。"),
	},
}
