package eden

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

func WithMaxGo(maxGo int) Opt {
	return func(p *DocxProcessor) {
		p.maxGo = maxGo
	}
}

func WithLogger(logger *logger.DocxLogger) Opt {
	return func(p *DocxProcessor) {
		p.logger = logger
	}
}

func WithMaxToken(maxToken int) Opt {
	return func(p *DocxProcessor) {
		p.maxToken = maxToken
	}
}

// DocxProcessor DOCX 处理器
type DocxProcessor struct {
	fromLang    string
	toLang      string
	f           *document.Document
	closeFunc   func() error
	fileName    string
	paraSet     []translate.Paragraph
	tranParaSet map[string]string
	maxToken    int

	inputPath   string
	outputDir   string
	maxGo       int
	process     translate.Translate
	langChecker lang.LanguageChecker
	logger      *logger.DocxLogger

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
	if p.maxToken <= 0 {
		p.maxToken = 1 << 16 // 默认最大文字数 65536
	}

	p.paraSet = make([]translate.Paragraph, 0)
	p.tranParaSet = make(map[string]string, 0)
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
	totalCount := 0
	paragraphCount := 0
	segmentCount := 0
	tableCount := len(p.f.Tables())
	paraTmp := make(translate.Paragraph, 0, 1<<8)
	caluText := strings.Builder{}

	addText := func(s *strings.Builder, str string, clearOld bool) {
		if clearOld {
			s.Reset()
		}
		s.WriteString(str)
		s.WriteString(translate.Seq)
	}

	paragraphs := p.f.Paragraphs()
	for idx, paragraph := range paragraphs {
		paragraphCount++
		runs := paragraph.Runs()
		if len(runs) == 0 {
			continue
		}

		for _, r := range runs {
			text := r.Text()

			// 语言检查
			if trimText := strings.TrimSpace(text); strings.TrimSpace(trimText) == "" || (p.langChecker != nil && p.langChecker.Check(trimText)) {
				p.logger.Info(fmt.Sprintf("忽略文本块[%v]", text))
				continue
			}

			segmentCount++
			totalCount += len([]rune(text))

			if p.logger != nil {
				p.logger.LogParagraphProcessing(segmentCount, text, true)
			}
			addText(&caluText, text, false)

			// 检查长度
			if len([]rune(caluText.String())) > p.maxToken && len(paraTmp) > 0 {
				p.paraSet = append(p.paraSet, paraTmp)
				paraTmp = make(translate.Paragraph, 0)
				addText(&caluText, text, true)
			}
			paraTmp = append(paraTmp, text)
		}

		// 最后
		if idx == len(paragraphs)-1 && len(paraTmp) > 0 {
			p.paraSet = append(p.paraSet, paraTmp)
			paraTmp = make(translate.Paragraph, 0)
			caluText.Reset()
		}
	}

	if p.logger != nil {
		p.logger.LogTextExtraction(paragraphCount, segmentCount, tableCount, totalCount)
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
		p.logger.Info("开始翻译%d个分块", len(p.paraSet))
	}

	pool, _ := ants.NewPool(p.maxGo,
		ants.WithMaxBlockingTasks(1<<20),
		ants.WithPreAlloc(true),
		ants.WithExpiryDuration(1))

	for k, paragraph := range p.paraSet {
		paraIdx := k
		paraCopy := paragraph
		paraStr := strings.Join(paraCopy, "|")
		if p.langChecker != nil && p.langChecker.Check(paraStr) {
			if p.logger != nil {
				p.logger.Info("翻译跳过:%d [%s]", k, paraStr)
			}

			p.rw.Lock()
			p.tranParaSet = combineMap(p.tranParaSet, fillMap(paraCopy))
			p.rw.Unlock()
			continue
		}

		p.wg.Add(1)
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
					return
				} else {
					translatedText := strings.Join(t, "|")
					p.logger.LogTranslationResponse(paraIdx, true, translatedText, nil)
				}
			}

			p.rw.Lock()
			p.tranParaSet = combineMap(p.tranParaSet, fillMap(paraCopy, t))
			p.rw.Unlock()
		})
	}

	p.wg.Wait()
	pool.Release()
}

// WriteChanges 将处理后的内容写回 DOCX 文件
func (p *DocxProcessor) WriteChanges() {
	if p.logger != nil {
		p.logger.Info("开始将翻译结果写回文档")
	}

	paragraphs := p.f.Paragraphs()
	for _, paragraph := range paragraphs {
		runs := paragraph.Runs()
		if len(runs) == 0 {
			continue
		}

		for _, r := range runs {
			k := r.Text()

			if tranStr, ok := p.tranParaSet[k]; ok {
				r.ClearContent()
				r.AddText(tranStr)
			}
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

		// 记录翻译结束
		p.logger.LogTranslationEnd(outPath, err == nil, time.Since(startTime))
	}

	return err
}

func fillMap(src ...[]string) map[string]string {
	m := map[string]string{}
	if len(src) == 0 {
		return m
	}

	key := src[0]
	val := src[0]
	if len(src) > 1 {
		val = src[1]
	}
	if len(key) != len(val) {
		return m
	}

	for i, k := range key {
		m[k] = val[i]
	}
	return m
}

func combineMap(maps ...map[string]string) map[string]string {
	m := map[string]string{}
	for _, mm := range maps {
		for k, v := range mm {
			m[k] = v
		}
	}
	return m
}
