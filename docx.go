package eden

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gou-jjjj/eden/lang"
	"github.com/gou-jjjj/eden/translate"
	"github.com/gou-jjjj/unioffice/document"
	"github.com/panjf2000/ants"
)

var pool, _ = ants.NewPool(1<<10,
	ants.WithMaxBlockingTasks(1<<20),
	ants.WithPreAlloc(true),
	ants.WithExpiryDuration(1))

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
	process     translate.Translate
	langChecker lang.LanguageChecker
	elog        *slog.Logger

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

	if p.maxToken <= 0 {
		p.maxToken = 1 << 20
	}

	p.paraSet = make([]translate.Paragraph, 0)
	p.tranParaSet = make(map[string]string, 0)
	//p.langChecker = lang.LangMapChecks[p.toLang]
	p.fileName = strings.Split(filepath.Base(p.inputPath), ".")[0]
	_, err := os.Stat(p.outputDir)
	if err != nil {
		if os.IsNotExist(err) {
			_ = os.MkdirAll(p.outputDir, os.ModePerm)
		}
	}

	// 初始化日志记录器
	if p.elog == nil {
		p.elog = slog.Default()
	}

	return p
}

// LoadFile 从 DOCX 文件中加载文档
func (p *DocxProcessor) LoadFile() error {
	if p.inputPath == "" {
		err := fmt.Errorf("input path is required")
		return err
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
	totalCharCnt := 0
	segmentCount := 0
	tableCount := len(p.f.Tables())

	paragraphs := p.f.Paragraphs()
	for idx, paragraph := range paragraphs {
		runs := paragraph.Runs()
		paraTmp := make(translate.Paragraph, 0, 1<<8)
		caluText := strings.Builder{}
		charCnt := 0

		_ = p.handleRuns(runs, func(i int, text string) string {
			paraTmp = append(paraTmp, text)
			caluText.WriteString(text)
			charCnt += len([]rune(text))
			return text
		})
		p.paraSet = append(p.paraSet, paraTmp)
		segmentCount += len(paraTmp)
		totalCharCnt += charCnt
		p.elog.Info("处理段落[%d], 字符个数:[%d], 文字块个数:[%d],",
			idx, charCnt, len(paraTmp))
		p.elog.Debug("处理段落[%d],文字内容[%s]", idx, caluText.String())
	}

	p.elog.Info("文本提取完成")
	p.elog.Info("总文字数量: %d", totalCharCnt)
	p.elog.Info("文本块数量: %d", segmentCount)
	p.elog.Info("段落数量: %d", len(paragraphs))
	p.elog.Info("表格数量: %d", tableCount)

	return nil
}

func (p *DocxProcessor) handleRuns(runs []document.Run, f func(int, string) string) []document.Run {
	for i, r := range runs {
		text := r.Text()
		text = f(i, text)
		runs[i].AddText(text)
	}
	return runs
}

// ProcessText 处理文本内容
func (p *DocxProcessor) ProcessText() {
	if p.process == nil {
		if p.elog != nil {
			p.elog.Warn("没有设置翻译处理器，跳过翻译")
		}
		return // 如果没有处理函数，返回原文本
	}

	if len(p.paraSet) == 0 {
		if p.elog != nil {
			p.elog.Info("没有需要翻译的段落")
		}
		return
	}

	if p.elog != nil {
		p.elog.Info("开始翻译%d个分块", len(p.paraSet))
	}

	plen := len(p.paraSet)
	for i := 0; i < plen; i++ {
		paraIdx := i
		paraCopy := p.paraSet[i]
		startIdx := 0
		endIdx := 0

		if i == 0 && plen > 1 {
			endIdx = len(paraCopy)
			paraCopy = append(paraCopy, p.paraSet[i+1]...)
		} else if i == len(p.paraSet)-1 && plen > 1 {
			paraCopy = append(p.paraSet[i-1], paraCopy...)
			startIdx = len(p.paraSet[i-1])
			endIdx = len(paraCopy)
		} else {
			paraCopy = append(p.paraSet[i-1], paraCopy...)
			startIdx = len(p.paraSet[i-1])
			endIdx = len(paraCopy)
			paraCopy = append(paraCopy, p.paraSet[i+1]...)
		}

		p.wg.Add(1)
		_ = pool.Submit(func() {
			defer p.wg.Done()

			// 记录翻译请求
			if p.elog != nil {
				//text := strings.Join(paraCopy, " ")
				//p.elog.LogTranslationRequest(paraIdx, p.fromLang, p.toLang, text)
			}

			t, err := p.process.T(&translate.TranReq{
				From:  p.fromLang,
				To:    p.toLang,
				Paras: paraCopy,
			})

			// 记录翻译响应
			if err != nil {
				p.elog.Error("翻译段落[%d]失败: %v", paraIdx, err)
				return
			}
			p.rw.Lock()
			p.tranParaSet = combineMap(p.tranParaSet, fillMap(p.paraSet[i], t[startIdx:endIdx]))
			p.rw.Unlock()
		})
	}

	p.wg.Wait()
}

// WriteChanges 将处理后的内容写回 DOCX 文件
func (p *DocxProcessor) WriteChanges() {
	if p.elog != nil {
		p.elog.Info("开始将翻译结果写回文档")
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

	if p.elog != nil {
		p.elog.Info("翻译结果写回完成")
	}
}

// Process 执行完整的 DOCX 处理流程
func (p *DocxProcessor) Process() error {
	// 记录翻译开始
	p.elog.Info("翻译器:%+v,文件名字:%+v", p.process.Name(), p.fileName)
	if p.closeFunc == nil {
		defer p.closeFunc()
	}
	// 1. 加载文件
	if err := p.LoadFile(); err != nil {
		return err
	}

	// 2. 提取文本
	if err := p.ExtractText(); err != nil {
		return err
	}

	//3. 处理文本
	p.ProcessText()

	//// 4. 写回修改
	//p.WriteChanges()

	// 5. 保存文件
	//outPath := path.Join(p.outputDir, fmt.Sprintf("%s_%s.docx", p.fileName, lang.LangNames[p.toLang]))
	//err := p.f.SaveToFile(outPath)

	// 记录文件保存结果
	//if p.elog != nil {
	//	p.elog.LogFileSave(err == nil, outPath, err)
	//
	//	// 记录翻译结束
	//	p.elog.LogTranslationEnd(outPath, err == nil, time.Since(startTime))
	//}

	return nil
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
