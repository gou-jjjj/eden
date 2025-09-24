package translate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gou-jjjj/eden/prompt"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

const (
	ZhiPu      = "zhipu"
	GithubFree = "githubfree"
	OpenRouter = "openrouter"
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

func NewOpenai(url, key, model string) *TranOpenai {
	return &TranOpenai{
		url:   url,
		key:   key,
		model: model,
	}
}

func (t *TranOpenai) T(req *TranReq) ([]Paragraph, error) {
	ctx := context.Background()
	llm, err := openai.New(
		openai.WithBaseURL(t.url),
		openai.WithModel(t.model),
		openai.WithToken(t.key),
		openai.WithAPIType(openai.APITypeOpenAI),
		//openai.WithResponseFormat(openai.ResponseFormatJSON),
	)
	if err != nil {
		return nil, err
	}

	msg, _ := json.Marshal(req.Paras)
	content := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, prompt.TranslatePrompt(req.From, req.To)),
		llms.TextParts(llms.ChatMessageTypeHuman, "[[\"The following code prints Hello World:\",\"```python print('Hello World')```\"],[\"End of example.\"]]"),
		llms.TextParts(llms.ChatMessageTypeAI, "[[\"以下代码打印 Hello World：\",\"```python print('Hello World')```\"],[\"示例结束。\"]]"),
		llms.TextParts(llms.ChatMessageTypeHuman, string(msg)),
	}
	generateContent, err := llm.GenerateContent(ctx, content)
	if err != nil {
		return nil, err
	}

	_ = os.WriteFile(fmt.Sprintf("./out_openai_%d.txt", time.Now().Unix()), []byte(generateContent.Choices[0].Content), 0644)

	var res []Paragraph
	err = json.Unmarshal([]byte(generateContent.Choices[0].Content), &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}
