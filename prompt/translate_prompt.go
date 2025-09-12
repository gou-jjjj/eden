package prompt

import (
	_ "embed"
	"strings"
	"text/template"
)

//go:embed translate_prompt.md
var translatePrompt string

func TranslatePrompt(fromLang, toLang, paraLen string) string {
	// 解析模板
	tmpl, err := template.New("translatePrompt").Parse(translatePrompt)
	if err != nil {
		panic(err)
	}

	data := map[string]interface{}{
		"fromLang": fromLang,
		"toLang":   toLang,
		"paraLen":  paraLen,
	}

	// 执行模板，把渲染结果输出到标准输出
	builder := strings.Builder{}
	err = tmpl.Execute(&builder, data)
	if err != nil {
		panic(err)
	}

	return builder.String()
}
