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
