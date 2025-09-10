package main

import (
	"log"
	"os"
	"testing"

	"github.com/gou-jjjj/eden/translate"
)

func debugDelete() {
	err := os.RemoveAll("./out")
	if err != nil {
		log.Fatal(err)
	}
}

func TestNewDocxProcessor(t *testing.T) {
	debugDelete()

	pr := NewDocxProcessor(WithLang(EN, ZH),
		WithInput("./file_examples/zlobinski2011.docx"),
		WithOutput("./out"),
		WithProcessFunc(translate.NewMockTran()),
		WithMaxWords(3000),
		WithMaxGo(1))

	err := pr.Process()
	if err != nil {
		t.Error(err)
	}
}
