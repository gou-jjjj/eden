package translate

import (
	"context"
	"fmt"
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
	Ollama     = "ollama"

	Seq = "\n------------\n"
)

var OpenaiModelList = map[string]struct {
	Url   string
	Key   string
	Model string
}{
	ZhiPu:      {"https://open.bigmodel.cn/api/paas/v4", "23c650b6b73d4b1b80500699edcbf87c.qU9lcBTeGfku2gKz", "glm-4-plus"},
	GithubFree: {"https://api.chatanywhere.tech", "sk-vINYqBzbzrhdsFxZCO7MSSEvHL8tPradBhl77tLmWmEoTXs5", "deepseek-v3"},
	OpenRouter: {"https://openrouter.ai/api/v1", "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", "x-ai/grok-4-fast:free"},
	AliBaBa:    {"https://dashscope.aliyuncs.com/compatible-mode/v1", "sk-227cf58d893d4a689e82d2b8eb8f3564", "qwen-plus"},
	Ollama: {Url: "http://localhost:11434", // Ollama 默认地址
		Model: "qwen3:30b"},
}

type TranOpenai struct {
	url    string
	key    string
	model  string
	back   *TranOpenai
	logger logger.Logger
}

func NewOpenai(llmSource string, backTranOpenai ...*TranOpenai) *TranOpenai {
	s, ok := OpenaiModelList[llmSource]
	if !ok {
		return nil
	}
	return &TranOpenai{
		url:   s.Url,
		key:   s.Key,
		model: s.Model,
		back: func() *TranOpenai {
			if len(backTranOpenai) > 0 {
				return backTranOpenai[0]
			}
			return nil
		}(),
	}
}

func NewOpenaiWithLogger(llmSource string, logger logger.Logger, backTranOpenai ...*TranOpenai) *TranOpenai {
	s, ok := OpenaiModelList[llmSource]
	if !ok {
		return nil
	}
	return &TranOpenai{
		url:    s.Url,
		key:    s.Key,
		model:  s.Model,
		logger: logger,
		back: func() *TranOpenai {
			if len(backTranOpenai) > 0 {
				return backTranOpenai[0]
			}
			return nil
		}(),
	}
}

// 仅执行一次翻译，不再包含任何重试
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
	content := append([]llms.MessageContent{
		llms.TextParts(
			llms.ChatMessageTypeSystem,
			prompt.TranslatePrompt(req.From, req.To, len(req.Paras)),
		)},
		msgs...,
	)
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
		t.addLog(req, res)

	}

	return res, nil
}

func (t *TranOpenai) addLog(req *TranReq, res []string) {
	if t.logger != nil {
		s := strings.Builder{}
		for i := 0; i < max(len(req.Paras), len(res)); i++ {
			if i < len(req.Paras) {
				s.WriteString(fmt.Sprintf("[%s]->", req.Paras[i]))
			} else {
				s.WriteString("[] ")
			}
			if i < len(res) {
				s.WriteString(fmt.Sprintf("[%s] ", res[i]))
			} else {
				s.WriteString("[] ")
			}
			s.WriteString(Seq)
		}

		_ = os.WriteFile(
			fmt.Sprintf("error_resp_%d.log", time.Now().Unix()),
			[]byte(s.String()),
			0644,
		)
		t.logger.Warn(
			"翻译结果段落数与请求段落数不匹配，可能存在部分翻译丢失，req:%d, res:%d",
			len(req.Paras), len(res),
		)
	}
}

// 不再带重试，只执行一次，失败后自动使用备用翻译器
func (t *TranOpenai) T(req *TranReq) (Paragraph, error) {
	result, err := t.performTranslation(req)
	if err == nil {
		return result, nil
	}

	if t.logger != nil {
		t.logger.Warn("主翻译器失败: %v", err)
	}

	// 使用备用翻译器
	if t.back != nil {
		if t.logger != nil {
			t.logger.Info("尝试备用翻译器")
		}
		return t.back.T(req)
	}

	return nil, err
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
