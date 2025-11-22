package translate

import (
	"context"
	"fmt"
	"testing"

	"github.com/tmc/langchaingo/llms/ollama"
)

func TestName(t *testing.T) {
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

	fmt.Printf("%v\n", call)
}

func TestNewOpenai(t *testing.T) {
	llm, err := ollama.New(
		ollama.WithModel("qwen3:30b"))
	if err != nil {
		t.Fatal(err.Error())
	}
	o := NewOpenai(context.Background(), nil, 3, llm)
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
	s, err = o.T(&TranReq{
		From: "英语",
		To:   "法语",
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
	s, err = o.T(&TranReq{
		From: "英语",
		To:   "日语",
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
