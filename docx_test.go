package eden

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/gou-jjjj/eden/lang"
	"github.com/gou-jjjj/eden/prompt"
	"github.com/gou-jjjj/eden/translate"
	"github.com/gou-jjjj/unioffice/document"
	"github.com/tmc/langchaingo/llms/openai"
)

func debugDelete() {
	err := os.RemoveAll("./out")
	if err != nil {
		log.Fatal(err)
	}
}

func TestNewDocxProcessor(t *testing.T) {
	llm, err := openai.New(
		openai.WithModel("qwen-plus"),
		openai.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),
		openai.WithToken("sk-227cf58d893d4a689e82d2b8eb8f3564"),
	)
	o := translate.NewAiTran(context.Background(), nil, 3, llm)

	pr := NewDocxProcessor(
		WithInput("/Users/calvin/go/src/eden/file_examples/dxusercu_e43caac4e7a606e6f290e3718d67ce21.docx"),
		WithOutput("./out"),
		WithLang(lang.EN),
		//WithProcessFunc(translate.NewMockTran()),
		WithProcessFunc(o),
		WithMaxToken(100))

	err = pr.Process()
	if err != nil {
		t.Error(err)
	}
}

func TestContent(t *testing.T) {
	const path = "/Users/calvin/go/src/eden/file_examples/dxusercu_e43caac4e7a606e6f290e3718d67ce21.docx"
	doc, err := document.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()

	// 检查文档基本信息
	t.Logf("段落数量: %d", len(doc.Paragraphs()))
	t.Logf("表格数量: %d", len(doc.Tables()))

	// 检查段落详情
	for _, para := range doc.Paragraphs() {
		for _, run := range para.Runs() {
			fmt.Print(run.Text())
		}
		fmt.Println()
	}
}

func TestPrompt(t *testing.T) {
	translatePrompt := prompt.TranslatePrompt("english", "chinese")
	t.Logf("%v\n", translatePrompt)
}
