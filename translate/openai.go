package translate

import (
	"context"
	"encoding/json"

	"github.com/gou-jjjj/eden/prompt"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

const (
	apiURL    = "https://open.bigmodel.cn/api/paas/v4"
	apiKey    = "23c650b6b73d4b1b80500699edcbf87c.qU9lcBTeGfku2gKz" // 请替换为您的实际 API 密钥
	modelName = "glm-4-plus"

	//	prompt = `你是一个专业的翻译助手，擅长将文本从一种语言翻译到另一种语言。
	//规则：
	//1. 返回结果必须是 JSON 格式，结构为 []Content，其中 Content 包含 id 和 data 字段。
	//2. 保留输入中的所有非文本字符（如标点符号、数字、特殊字符）。
	//3. 翻译时保持 id 不变，仅翻译 data 字段的内容。
	//4. 严格按照目标语言翻译，不要遗漏或添加内容。
	//示例输入: {"from":"zh","to":"en","data":[{"id":1,"data":"你好"},{"id":2,"data":"1.你是谁*"}]}
	//示例输出: [{"id":1,"data":"hello"},{"id":2,"data":"1.who are you*"}]`
)

type TranOpenai struct {
}

func NewOpenai() *TranOpenai {
	return &TranOpenai{}
}

func (t *TranOpenai) T(req *TranReq) ([]Content, error) {
	ctx := context.Background()
	llm, err := openai.New(
		openai.WithBaseURL(apiURL),
		openai.WithModel(modelName),
		openai.WithToken(apiKey),
		openai.WithAPIType(openai.APITypeOpenAI),
		openai.WithResponseFormat(openai.ResponseFormatJSON),
	)
	if err != nil {
		return nil, err
	}

	msg, _ := json.Marshal(req)
	content := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, prompt.TranslatePrompt(req.From, req.To)),
		llms.TextParts(llms.ChatMessageTypeHuman, string(msg)),
	}
	generateContent, err := llm.GenerateContent(ctx, content)
	if err != nil {
		return nil, err
	}

	var res []Content
	err = json.Unmarshal([]byte(generateContent.Choices[0].Content), &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}
