package main

import (
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"unicode/utf8"

	"github.com/gou-jjjj/eden/translate"
)

// Document 代表 docx 文件中的 XML 文档结构
type Document struct {
	XMLName xml.Name `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main document"`
	Body    Body     `xml:"body"`
}

// Body 包含段落列表
type Body struct {
	Paragraphs []Paragraph `xml:"p"`
}

// Paragraph 包含文本运行列表
type Paragraph struct {
	Runs []Run `xml:"r"`
}

// Run 包含文本内容
type Run struct {
	Text []Text `xml:"t"`
}

// Text 包含实际文本值
type Text struct {
	Value string `xml:",chardata"`
}

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

func WithLangChecker(checker LanguageChecker) Opt {
	return func(p *DocxProcessor) {
		p.langChecker = checker
	}
}

func WithMaxWords(maxWorks int) Opt {
	return func(p *DocxProcessor) {
		p.maxWords = maxWorks
	}
}

func WithMaxGo(maxGo int) Opt {
	return func(p *DocxProcessor) {
		p.maxGo = maxGo
	}
}

// DocxProcessor DOCX 处理器
type DocxProcessor struct {
	inputPath   string
	outputDir   string
	outputPath  string
	fromLang    string
	toLang      string
	maxWords    int
	maxGo       int
	process     translate.Translate
	langChecker LanguageChecker

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

	p.langChecker = LangMapChecks[p.fromLang]

	return p
}

// Copy 复制 DOCX 文件到指定目录
func (p *DocxProcessor) Copy() (string, error) {
	if p.inputPath == "" {
		return "", fmt.Errorf("input path is required")
	}

	ext := filepath.Ext(p.inputPath)
	if ext != ".docx" {
		return "", fmt.Errorf("input file is not a .docx file")
	}

	if p.outputDir == "" {
		return "", fmt.Errorf("output directory is required")
	}

	// 确保输出目录存在
	if err := os.MkdirAll(p.outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %v", err)
	}

	inputFile, err := os.Open(p.inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to open input file: %v", err)
	}
	defer inputFile.Close()

	p.outputPath = filepath.Join(p.outputDir, filepath.Base(p.inputPath))

	outputFile, err := os.Create(p.outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %v", err)
	}
	defer outputFile.Close()

	_, err = io.Copy(outputFile, inputFile)
	if err != nil {
		return "", fmt.Errorf("failed to copy file: %v", err)
	}

	return p.outputPath, nil
}

// ExtractText 从 DOCX 文件中提取文本内容
func (p *DocxProcessor) ExtractText() (Document, map[string]string, error) {
	if p.outputPath == "" {
		return Document{}, nil, fmt.Errorf("output path is required")
	}

	reader, err := zip.OpenReader(p.outputPath)
	if err != nil {
		return Document{}, nil, fmt.Errorf("failed to open docx file: %v", err)
	}
	defer reader.Close()

	var doc Document
	textMap := make(map[string]string)

	for _, f := range reader.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return Document{}, nil, fmt.Errorf("failed to open %s: %v", f.Name, err)
			}
			defer rc.Close()

			data, err := io.ReadAll(rc)
			if err != nil {
				return Document{}, nil, fmt.Errorf("failed to read %s: %v", f.Name, err)
			}

			if err := xml.Unmarshal(data, &doc); err != nil {
				return Document{}, nil, fmt.Errorf("failed to parse document.xml: %v", err)
			}

			// 使用 LangChecker 函数检查文本是否需要处理
			check := p.langChecker
			if check == nil {
				// 默认检查函数，处理所有非空文本
				check = ChineseChecker{}
			}

			for i := range doc.Body.Paragraphs {
				for j := range doc.Body.Paragraphs[i].Runs {
					for _, text := range doc.Body.Paragraphs[i].Runs[j].Text {
						if check.Check(text.Value) {
							textMap[text.Value] = text.Value
						}
					}
				}
			}
			return doc, textMap, nil
		}
	}
	return Document{}, nil, fmt.Errorf("word/document.xml not found in docx file")
}

// ProcessText 处理文本内容
func (p *DocxProcessor) ProcessText(textMap map[string]string) map[string]string {
	if p.process == nil {
		return textMap // 如果没有处理函数，返回原文本
	}

	contents := make([]translate.Content, 0, len(textMap))
	tmp := make(map[int]string, len(textMap))
	id := 0
	for key := range textMap {
		id++
		contents = append(contents, translate.Content{
			Id:   id,
			Data: key,
		})
		tmp[id] = key
	}

	reqCont := make([]translate.Content, 0, len(contents))
	maxCh := make(chan struct{}, p.maxGo)

	toTran := func(r *translate.TranReq) {
		defer func() {
			<-maxCh
			p.wg.Done()
		}()

		t, err := p.process.T(r)
		if err != nil {
			log.Printf("failed to process text: %v", err)
			return
		}

		for _, res := range t {
			p.rw.Lock()
			if original, exists := tmp[res.Id]; exists {
				textMap[original] = res.Data
			} else {
				log.Printf("id %d not found in tmp map", res.Id)
			}
			p.rw.Unlock()
		}
	}

	for i, content := range contents {
		addCont := append(reqCont, content)
		contentLen, _ := json.Marshal(translate.TranReq{
			From: p.fromLang,
			To:   p.toLang,
			Data: addCont,
		})

		if utf8.RuneCountInString(string(contentLen)) > p.maxWords {
			singReq := translate.TranReq{
				From: p.fromLang,
				To:   p.toLang,
				Data: reqCont,
			}
			if i == len(contents)-1 {
				singReq.Data = addCont
			}
			reqCont = make([]translate.Content, 0, len(addCont))

			maxCh <- struct{}{}
			p.wg.Add(1)
			go toTran(&singReq)
		}

		reqCont = append(reqCont, content)
	}

	if len(reqCont) > 0 {
		maxCh <- struct{}{}
		p.wg.Add(1)
		go toTran(&translate.TranReq{
			From: p.fromLang,
			To:   p.toLang,
			Data: reqCont,
		})
	}

	p.wg.Wait()

	return textMap
}

// WriteChanges 将处理后的内容写回 DOCX 文件
func (p *DocxProcessor) WriteChanges(doc Document, textMap map[string]string) error {
	if p.outputPath == "" {
		return fmt.Errorf("output path is required")
	}

	// 打开现有的 docx 文件进行读写
	reader, err := zip.OpenReader(p.outputPath)
	if err != nil {
		return fmt.Errorf("failed to open docx file for writing: %v", err)
	}
	defer reader.Close()

	// 创建临时文件用于写入新 docx
	tempFile, err := os.CreateTemp("", "docx_*.docx")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name()) // 清理临时文件

	zipWriter := zip.NewWriter(tempFile)
	defer zipWriter.Close()

	// 恢复文档中的文本
	for i := range doc.Body.Paragraphs {
		for j := range doc.Body.Paragraphs[i].Runs {
			for k := range doc.Body.Paragraphs[i].Runs[j].Text {
				if text, exists := textMap[doc.Body.Paragraphs[i].Runs[j].Text[k].Value]; exists {
					doc.Body.Paragraphs[i].Runs[j].Text[k].Value = text
				} else if p.langChecker != nil && p.langChecker.Check(doc.Body.Paragraphs[i].Runs[j].Text[k].Value) {
					log.Printf("text [%s] not found in textMap", doc.Body.Paragraphs[i].Runs[j].Text[k].Value)
				}
			}
		}
	}

	// 序列化修改后的 XML
	documentXML, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal modified XML: %v", err)
	}

	// 写入新 docx 文件
	for _, f := range reader.File {
		writer, err := zipWriter.Create(f.Name)
		if err != nil {
			return fmt.Errorf("failed to create zip entry %s: %v", f.Name, err)
		}

		if f.Name == "word/document.xml" {
			// 写入修改后的 document.xml
			_, err = writer.Write([]byte(xml.Header + string(documentXML)))
			if err != nil {
				return fmt.Errorf("failed to write modified document.xml: %v", err)
			}
		} else {
			// 复制其他文件
			rc, err := f.Open()
			if err != nil {
				return fmt.Errorf("failed to open %s: %v", f.Name, err)
			}
			_, err = io.Copy(writer, rc)
			rc.Close()
			if err != nil {
				return fmt.Errorf("failed to copy %s: %v", f.Name, err)
			}
		}
	}

	// 关闭 zipWriter 以确保数据写入
	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close zip writer: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %v", err)
	}

	// 覆盖原始文件
	if err := os.Rename(tempFile.Name(), p.outputPath); err != nil {
		return fmt.Errorf("failed to rename temp file to %s: %v", p.outputPath, err)
	}

	return nil
}

// Process 执行完整的 DOCX 处理流程
func (p *DocxProcessor) Process() error {
	// 1. 复制文件
	if _, err := p.Copy(); err != nil {
		return err
	}
	log.Printf("new file path: %s ", p.outputPath)
	// 2. 提取文本
	doc, textMap, err := p.ExtractText()
	if err != nil {
		return err
	}

	// 3. 处理文本
	processedMap := p.ProcessText(textMap)

	// 4. 写回修改
	return p.WriteChanges(doc, processedMap)
}
