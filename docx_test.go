package eden

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/gou-jjjj/eden/lang"
	"github.com/gou-jjjj/eden/prompt"
	"github.com/gou-jjjj/eden/translate"
	"github.com/gou-jjjj/unioffice/document"
)

func debugDelete() {
	err := os.RemoveAll("./out")
	if err != nil {
		log.Fatal(err)
	}
}

func TestNewDocxProcessor(t *testing.T) {

	pr := NewDocxProcessor(
		WithInput("C:\\Users\\Administrator\\go\\src\\eden\\file_examples\\Docx4j_GettingStarted.docx"),
		WithOutput("./out"),
		WithLang(lang.ZH),
		//WithProcessFunc(translate.NewOpenaiWithLogger(translate.Ollama, newLogger, open)),
		WithProcessFunc(translate.NewMockTran()),
		WithMaxToken(100))

	err := pr.Process()
	if err != nil {
		t.Error(err)
	}

}

func TestContent(t *testing.T) {
	const path = "/Users/calvin/go/src/eden/file_examples/Docx4j_GettingStarted.docx"
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

func TestLogging(t *testing.T) {
	debugDelete()

	pr := NewDocxProcessor(WithLang(lang.All, lang.EN),
		WithInput("C:\\Users\\Administrator\\go\\src\\eden\\file_examples\\dxusercu_e43caac4e7a606e6f290e3718d67ce21.docx"),
		WithOutput("./out"),
		WithProcessFunc(translate.NewMockTran()))

	err := pr.Process()
	if err != nil {
		t.Error(err)
	}
}
