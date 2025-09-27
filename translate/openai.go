package translate

import (
	"context"
	"fmt"
	"strings"

	"github.com/gou-jjjj/eden/lang"
	"github.com/gou-jjjj/eden/prompt"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

const (
	ZhiPu      = "zhipu"
	GithubFree = "githubfree"
	OpenRouter = "openrouter"

	seq = "\n---\n"
)

var OpenaiModelList = map[string]struct {
	Url   string
	Key   string
	Model string
}{
	ZhiPu:      {"https://open.bigmodel.cn/api/paas/v4", "23c650b6b73d4b1b80500699edcbf87c.qU9lcBTeGfku2gKz", "glm-4-plus"},
	GithubFree: {"https://api.chatanywhere.tech", "sk-vINYqBzbzrhdsFxZCO7MSSEvHL8tPradBhl77tLmWmEoTXs5", "deepseek-v3"},
	OpenRouter: {"https://openrouter.ai/api/v1", "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", "x-ai/grok-4-fast:free"},
}

type TranOpenai struct {
	url   string
	key   string
	model string
}

func NewOpenai(llmSource string) *TranOpenai {
	s, ok := OpenaiModelList[llmSource]
	if !ok {
		return nil
	}

	return &TranOpenai{
		url:   s.Url,
		key:   s.Key,
		model: s.Model,
	}
}

func (t *TranOpenai) T(req *TranReq) (Paragraph, error) {
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
	content := append([]llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, prompt.TranslatePrompt(req.From, req.To))}, msgs...)
	content = append(content, llms.TextParts(llms.ChatMessageTypeHuman, strings.Join(req.Paras, seq)))
	generateContent, err := llm.GenerateContent(ctx, content)
	if err != nil {
		return nil, err
	}

	res := strings.Split(generateContent.Choices[0].Content, seq)
	return res, nil
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
