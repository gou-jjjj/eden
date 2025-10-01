package prompt

import (
	_ "embed"
	"strings"
	"text/template"
)

//go:embed translate_prompt.md
var translatePrompt string

//go:embed translate_single_prompt.md
var translateSinglePrompt string

func TranslatePrompt(fromLang, toLang string, segLen ...int) string {
	// 解析模板
	prompt := translatePrompt
	if len(segLen) >= 1 {
		switch segLen[0] {
		case 1:
			prompt = translateSinglePrompt
		default:
		}
	}

	tmpl, err := template.New("translatePrompt").Parse(prompt)
	if err != nil {
		panic(err)
	}

	data := map[string]interface{}{
		"fromLang": fromLang,
		"toLang":   toLang,
	}

	// 执行模板，把渲染结果输出到标准输出
	builder := strings.Builder{}
	err = tmpl.Execute(&builder, data)
	if err != nil {
		panic(err)
	}

	return builder.String()
}
