package translate

import (
	"context"
	"testing"

	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
)

func TestTest(t *testing.T) {
	llm, err := ollama.New(
		ollama.WithModel("qwen3:8b"),
	)
	if err != nil {
		t.Log(err.Error())
		return
	}

	call, err := llm.Call(context.Background(), "你是谁")
	if err != nil {
		t.Log(err.Error())
		return
	}

	t.Log(call)
}

func TestNewAiTran(t *testing.T) {
	llm, err := openai.New(
		openai.WithModel("qwen-plus"),
		openai.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),
		openai.WithToken("sk-227cf58d893d4a689e82d2b8eb8f3564"),
	)
	o := NewAiTran(context.Background(), nil, 3, llm)
	s, err := o.T(&TranReq{
		From: "英语",
		To:   "简体中文",
		Paras: []string{
			"Open an existing docx/pptx/xlsx document	11",
			"OpenXML concepts	12",
			"Specification versions	13",
			"Architecture	13",
			"Jaxb: marshalling and unmarshalling	15",
			"Parts List	16",
			"MainDocumentPart	18",
			"Samples	19",
		},
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(s)
}
