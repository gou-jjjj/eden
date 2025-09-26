package main

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gou-jjjj/eden/lang"
	"github.com/gou-jjjj/eden/translate"
	"github.com/gou-jjjj/unioffice/document"
	"github.com/panjf2000/ants"
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

func WithLang(from, to string) Opt {
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

func WithMaxGo(maxGo int) Opt {
	return func(p *DocxProcessor) {
		p.maxGo = maxGo
	}
}

// DocxProcessor DOCX 处理器
type DocxProcessor struct {
	fromLang    string
	toLang      string
	f           *document.Document
	closeFunc   func() error
	fileName    string
	paraSet     map[int]translate.Paragraph
	tranParaSet map[int]translate.Paragraph

	inputPath   string
	outputDir   string
	maxGo       int
	process     translate.Translate
	langChecker lang.LanguageChecker

	rw sync.Mutex
	wg sync.WaitGroup
}

// NewDocxProcessor 创建新的 DOCX 处理器
func NewDocxProcessor(opts ...Opt) *DocxProcessor {
	p := &DocxProcessor{}
	for _, opt := range opts {
		if opt != nil {
			opt(p)
		}
	}

	if p.maxGo <= 0 {
		p.maxGo = 1
	}

	p.paraSet = make(map[int]translate.Paragraph)
	p.tranParaSet = make(map[int]translate.Paragraph)
	p.langChecker = lang.LangMapChecks[p.toLang]
	p.fileName = filepath.Base(p.fileName)

	return p
}

// ExtractText 从 DOCX 文件中提取文本内容
func (p *DocxProcessor) LoadFile() error {
	if p.inputPath == "" {
		return fmt.Errorf("output path is required")
	}

	f, err := document.Open(p.inputPath)
	if err != nil {
		return err
	}

	p.f = f
	p.closeFunc = f.Close
	return nil
}

// ExtractText 从 DOCX 文件中提取文本内容
func (p *DocxProcessor) ExtractText() error {
	for idx, paragraph := range p.f.Paragraphs() {
		if len(paragraph.Runs()) == 0 {
			continue
		}

		paraTmp := make(translate.Paragraph, 0, len(paragraph.Runs()))
		needTran := false
		hasText := false
		for _, runs := range paragraph.Runs() {
			text := strings.TrimSpace(runs.Text())
			if text != "" {
				hasText = true
				if p.langChecker != nil && !p.langChecker.Check(text) {
					needTran = true
				}
			}
			paraTmp = append(paraTmp, runs.Text())
		}

		if needTran && hasText {
			p.paraSet[idx] = paraTmp
		}
	}

	return nil
}

// ProcessText 处理文本内容
func (p *DocxProcessor) ProcessText() {
	if p.process == nil {
		return // 如果没有处理函数，返回原文本
	}

	if len(p.paraSet) == 0 {
		return
	}

	pool, _ := ants.NewPool(p.maxGo,
		ants.WithMaxBlockingTasks(1<<20),
		ants.WithPreAlloc(true),
		ants.WithExpiryDuration(1))

	for k, paragraph := range p.paraSet {
		p.wg.Add(1)

		paraIdx := k
		paraCopy := make(translate.Paragraph, len(paragraph))
		copy(paraCopy, paragraph)

		_ = pool.Submit(func() {
			defer p.wg.Done()
			t, err := p.process.T(&translate.TranReq{
				From:  p.fromLang,
				To:    p.toLang,
				Paras: paraCopy,
			})
			if err != nil {
				return
			}

			p.rw.Lock()
			p.tranParaSet[paraIdx] = t
			p.rw.Unlock()
		})
	}

	p.wg.Wait()
	pool.Release()
}

// WriteChanges 将处理后的内容写回 DOCX 文件
func (p *DocxProcessor) WriteChanges() {
	for idx, tranSet := range p.tranParaSet {
		paragraph := p.f.Paragraphs()[idx]

		for tranIdx, run := range paragraph.Runs() {
			if tranIdx >= len(tranSet) {
				break
			}
			run.ClearContent()
			run.AddText(tranSet[tranIdx])
		}
	}
}

// Process 执行完整的 DOCX 处理流程
func (p *DocxProcessor) Process() error {
	defer func() {
		if p.closeFunc != nil {
			_ = p.closeFunc()
		}
	}()

	// 1. 复制文件
	if err := p.LoadFile(); err != nil {
		return err
	}

	// 2. 提取文本
	if err := p.ExtractText(); err != nil {
		return err
	}

	// 3. 处理文本
	p.ProcessText()

	// 4. 写回修改
	p.WriteChanges()

	outPath := path.Join(p.outputDir, fmt.Sprintf("%s/%s_%s.docx", p.outputDir, p.fileName, p.fromLang))
	return p.f.SaveToFile(outPath)
}
