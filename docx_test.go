package main

import (
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
	debugDelete()

	pr := NewDocxProcessor(WithLang(lang.EN, lang.ZH),
		WithInput("/Users/zyb/go/src/github.com/gou-jjjj/eden/file_examples/dxusercu_374966c9c16afa528c7cbed0ad763b12.docx"),
		WithOutput("./out"),
		WithLang(lang.All, lang.EN),
		WithProcessFunc(translate.NewOpenai(translate.OpenRouter)),
		WithMaxGo(3))

	err := pr.Process()
	if err != nil {
		t.Error(err)
	}
}

func TestName112(t *testing.T) {
	doc, err := document.Open("/Users/zyb/go/src/github.com/gou-jjjj/eden/file_examples/dxusercu_374966c9c16afa528c7cbed0ad763b12.docx")
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()

	// 检查文档基本信息
	t.Logf("段落数量: %d", len(doc.Paragraphs()))
	t.Logf("表格数量: %d", len(doc.Tables()))

	// 检查段落详情
	for _, para := range doc.Paragraphs() {
		paras := make([]string, 0, len(para.Runs()))
		c := false
		for _, run := range para.Runs() {
			if run.Text() != "" {
				c = true
			}
			paras = append(paras, run.Text())
			run.ClearContent()
		}

		if !c {
			continue
		}

		openai := translate.NewOpenai(translate.OpenRouter)
		paras, err = openai.T(&translate.TranReq{
			From:  lang.ZH,
			To:    lang.EN,
			Paras: paras,
		})
		if err != nil {
			t.Error(err)
			return
		}

		for i, run := range para.Runs() {
			run.AddText(paras[i])
		}
	}
	err = doc.SaveToFile("./out.docx")
	if err != nil {
		t.Error(err)
	}
}

func TestPrompt(t *testing.T) {
	translatePrompt := prompt.TranslatePrompt("english", "chinese")
	t.Logf("%v\n", translatePrompt)
}
