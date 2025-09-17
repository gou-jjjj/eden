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
	apiURL    = "https://open.bigmodel.cn/api/paas/v4"
	apiKey    = "23c650b6b73d4b1b80500699edcbf87c.qU9lcBTeGfku2gKz" // 请替换为您的实际 API 密钥
	modelName = "glm-4-plus"

	apiUrlV2    = "https://api.chatanywhere.tech"
	apiKeyV2    = "sk-vINYqBzbzrhdsFxZCO7MSSEvHL8tPradBhl77tLmWmEoTXs5" // 请替换为您的实际 API 密钥
	modelNameV2 = "deepseek-v3"
)

type TranOpenai struct {
}

func NewOpenai() *TranOpenai {
	return &TranOpenai{}
}

func (t *TranOpenai) T(req *TranReq) ([]Paragraph, error) {
	ctx := context.Background()
	llm, err := openai.New(
		openai.WithBaseURL(apiUrlV2),
		openai.WithModel(modelNameV2),
		openai.WithToken(apiKeyV2),
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
