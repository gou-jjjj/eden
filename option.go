package eden

import (
	"github.com/gou-jjjj/eden/lang"
	"github.com/gou-jjjj/eden/logger"
	"github.com/gou-jjjj/eden/translate"
)

// 选项函数类型
type Opt func(*DocxProcessor)

// 选项设置函数
func WithInput(path string) Opt {
	return func(p *DocxProcessor) {
		p.inputPath = path
	}
}

func WithOutput(dir string) Opt {
	return func(p *DocxProcessor) {
		p.outputDir = dir
	}
}

func WithLang(lg ...string) Opt {
	to := lang.EN
	if len(lg) == 1 {
		to = lg[0]
	}
	from := lang.All
	if len(lg) == 2 {
		from = lg[0]
		to = lg[1]
	}
	return func(p *DocxProcessor) {
		p.fromLang = from
		p.toLang = to
	}
}

func WithProcessFunc(f translate.Translate) Opt {
	return func(p *DocxProcessor) {
		p.process = f
	}
}

func WithLangChecker(checker lang.LanguageChecker) Opt {
	return func(p *DocxProcessor) {
		p.langChecker = checker
	}
}

func WithLogger(logger *logger.DocxLogger) Opt {
	return func(p *DocxProcessor) {
		p.elog = logger
	}
}

func WithMaxToken(maxToken int) Opt {
	return func(p *DocxProcessor) {
		p.maxToken = maxToken
	}
}
