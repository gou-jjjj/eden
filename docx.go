package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gou-jjjj/eden/lang"
	"github.com/gou-jjjj/eden/logger"
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

func WithLogger(logger *logger.Logger) Opt {
	return func(p *DocxProcessor) {
		p.logger = logger
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
	logger      *logger.Logger

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
	p.fileName = strings.Split(filepath.Base(p.inputPath), ".")[0]

	_, err := os.Stat(p.outputDir)
	if err != nil {
		if os.IsNotExist(err) {
			_ = os.MkdirAll(p.outputDir, os.ModePerm)
		}
	}

	// 初始化日志记录器
	if p.logger == nil {
		lg, err := logger.NewLogger(false, p.outputDir, p.fileName)
		if err != nil {
			fmt.Printf("警告: 无法创建日志记录器: %v\n", err)
		} else {
			p.logger = lg
		}
	}

	return p
}

// LoadFile 从 DOCX 文件中加载文档
func (p *DocxProcessor) LoadFile() error {
	if p.inputPath == "" {
		err := fmt.Errorf("input path is required")
		if p.logger != nil {
			p.logger.LogFileLoad(false, "", err)
		}
		return err
	}

	f, err := document.Open(p.inputPath)
	if err != nil {
		if p.logger != nil {
			p.logger.LogFileLoad(false, p.inputPath, err)
		}
		return err
	}

	p.f = f
	p.closeFunc = f.Close

	if p.logger != nil {
		p.logger.LogFileLoad(true, p.inputPath, nil)
	}
	return nil
}

// ExtractText 从 DOCX 文件中提取文本内容
func (p *DocxProcessor) ExtractText() error {
	paragraphCount := 0
	tableCount := len(p.f.Tables())

	for idx, paragraph := range p.f.Paragraphs() {
		if len(paragraph.Runs()) == 0 {
			continue
		}

		paraTmp := make(translate.Paragraph, 0, len(paragraph.Runs()))
		needTran := false
		hasText := false
		var originalText strings.Builder

		for _, runs := range paragraph.Runs() {
			text := strings.TrimSpace(runs.Text())
			if text != "" {
				hasText = true
				originalText.WriteString(text)
				if p.langChecker != nil && !p.langChecker.Check(text) {
					needTran = true
				}
			}
			paraTmp = append(paraTmp, runs.Text())
		}

		if hasText {
			paragraphCount++
			if p.logger != nil {
				p.logger.LogParagraphProcessing(idx, originalText.String(), needTran)
			}
		}

		if needTran && hasText {
			p.paraSet[idx] = paraTmp
		}
	}

	if p.logger != nil {
		p.logger.LogTextExtraction(paragraphCount, tableCount)
	}

	return nil
}

// ProcessText 处理文本内容
func (p *DocxProcessor) ProcessText() {
	if p.process == nil {
		if p.logger != nil {
			p.logger.Warn("没有设置翻译处理器，跳过翻译")
		}
		return // 如果没有处理函数，返回原文本
	}

	if len(p.paraSet) == 0 {
		if p.logger != nil {
			p.logger.Info("没有需要翻译的段落")
		}
		return
	}

	if p.logger != nil {
		p.logger.Info("开始翻译 %d 个段落", len(p.paraSet))
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

			// 记录翻译请求
			if p.logger != nil {
				text := strings.Join(paraCopy, " ")
				p.logger.LogTranslationRequest(paraIdx, p.fromLang, p.toLang, text)
			}

			t, err := p.process.T(&translate.TranReq{
				From:  p.fromLang,
				To:    p.toLang,
				Paras: paraCopy,
			})

			// 记录翻译响应
			if p.logger != nil {
				if err != nil {
					p.logger.LogTranslationResponse(paraIdx, false, "", err)
				} else {
					translatedText := strings.Join(t, "|")
					p.logger.LogTranslationResponse(paraIdx, true, translatedText, nil)
				}
			}

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

	if p.logger != nil {
		p.logger.Info("翻译完成，成功翻译 %d 个段落", len(p.tranParaSet))
	}
}

// WriteChanges 将处理后的内容写回 DOCX 文件
func (p *DocxProcessor) WriteChanges() {
	if p.logger != nil {
		p.logger.Info("开始将翻译结果写回文档")
	}

	for idx, tranSet := range p.tranParaSet {
		paragraph := p.f.Paragraphs()[idx]

		for tranIdx, run := range paragraph.Runs() {
			if tranIdx >= len(tranSet) {
				break
			}
			run.ClearContent()
			run.AddText(tranSet[tranIdx])
		}

		if p.logger != nil {
			p.logger.Debug("已更新段落 %d 的翻译内容", idx)
		}
	}

	if p.logger != nil {
		p.logger.Info("翻译结果写回完成")
	}
}

// Process 执行完整的 DOCX 处理流程
func (p *DocxProcessor) Process() error {
	startTime := time.Now()

	// 记录翻译开始
	if p.logger != nil {
		p.logger.LogTranslationStart(p.inputPath, p.fromLang, p.toLang)
		p.logger.Info("翻译器:%+v,文件名字:%+v,翻译最大并发数量:%+v",
			p.process.Name(), p.fileName, p.maxGo)
	}

	defer func() {
		if p.closeFunc != nil {
			_ = p.closeFunc()
		}

		// 关闭日志记录器
		if p.logger != nil {
			_ = p.logger.Close()
		}
	}()

	// 1. 加载文件
	if err := p.LoadFile(); err != nil {
		if p.logger != nil {
			p.logger.LogTranslationEnd("", false, time.Since(startTime))
		}
		return err
	}

	// 2. 提取文本
	if err := p.ExtractText(); err != nil {
		if p.logger != nil {
			p.logger.LogTranslationEnd("", false, time.Since(startTime))
		}
		return err
	}

	// 3. 处理文本
	p.ProcessText()

	// 4. 写回修改
	p.WriteChanges()

	// 5. 保存文件
	outPath := path.Join(p.outputDir, fmt.Sprintf("%s_%s.docx", p.fileName, lang.LangNames[p.toLang]))
	err := p.f.SaveToFile(outPath)

	// 记录文件保存结果
	if p.logger != nil {
		p.logger.LogFileSave(err == nil, outPath, err)

		// 记录统计信息
		totalParagraphs := len(p.paraSet)
		translatedParagraphs := len(p.tranParaSet)
		skippedParagraphs := totalParagraphs - translatedParagraphs
		p.logger.LogStatistics(totalParagraphs, translatedParagraphs, skippedParagraphs)

		// 记录翻译结束
		p.logger.LogTranslationEnd(outPath, err == nil, time.Since(startTime))
	}

	return err
}
