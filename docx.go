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
	"time"
	"unicode/utf8"

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
func (p *DocxProcessor) ExtractText() ([]translate.Paragraph, error) {
	if p.outputPath == "" {
		return nil, fmt.Errorf("output path is required")
	}

	reader, err := zip.OpenReader(p.outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open docx file: %v", err)
	}
	defer reader.Close()

	var data []translate.Paragraph
	for _, file := range reader.File {
		if file.Name == "word/document.xml" {
			rc, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open document.xml: %v", err)
			}
			defer rc.Close()

			decoder := xml.NewDecoder(rc)
			inParagraph := false
			var row []string
			for {
				token, err := decoder.Token()
				if err != nil {
					if err == io.EOF {
						break
					}
					return nil, fmt.Errorf("failed to decode XML token: %v", err)
				}

				switch se := token.(type) {
				case xml.StartElement:
					if se.Name.Local == "p" {
						inParagraph = true
					} else if inParagraph && se.Name.Local == "t" {
						// 读取文本内容
						var text struct {
							Content string `xml:",chardata"`
						}
						if err := decoder.DecodeElement(&text, &se); err == nil {
							if p.langChecker != nil && !p.langChecker.Check(text.Content) {
								row = append(row, text.Content)
							}
						}
					}
				case xml.EndElement:
					if se.Name.Local == "p" && inParagraph {
						if len(row) >= 0 {
							// 段落结束，写入内容并换行
							data = append(data, row)
							row = make([]string, 0)
						}
						inParagraph = false
					}
				}
			}
			break
		}
	}

	return data, nil
}

// ProcessText 处理文本内容
func (p *DocxProcessor) ProcessText(contents []translate.Paragraph) []translate.Paragraph {
	if p.process == nil {
		return contents // 如果没有处理函数，返回原文本
	}

	tmp := make(map[int]translate.Paragraph, len(contents))
	for id := range contents {
		tmp[id] = contents[id]
	}

	reqCont := make([]translate.Paragraph, 0, len(contents))
	maxCh := make(chan struct{}, p.maxGo)

	toTran := func(r *translate.TranReq, start int) {
		now := time.Now()
		log.Printf("start to process %d paragraphs", len(r.Paras))
		defer func() {
			log.Printf("processed %d paragraphs in %v", len(r.Paras), time.Since(now))
			<-maxCh
			p.wg.Done()
		}()

		t, err := p.process.T(r)
		if err != nil {
			log.Printf("failed to process text: %v", err)
			return
		}

		for idx, res := range t {
			contents[start+idx] = res
		}
	}

	for i, content := range contents {
		addCont := append(reqCont, content)
		contentLen, _ := json.Marshal(translate.TranReq{
			From:  p.fromLang,
			To:    p.toLang,
			Paras: addCont,
		})

		if utf8.RuneCountInString(string(contentLen)) > p.maxWords {
			singReq := translate.TranReq{
				From:  p.fromLang,
				To:    p.toLang,
				Paras: reqCont,
			}
			if i == len(contents)-1 {
				singReq.Paras = addCont
			}
			reqCont = make([]translate.Paragraph, 0, len(addCont))

			maxCh <- struct{}{}
			p.wg.Add(1)
			go toTran(&singReq, i-len(singReq.Paras))
		}

		reqCont = append(reqCont, content)
	}

	if len(reqCont) > 0 {
		maxCh <- struct{}{}
		p.wg.Add(1)
		go toTran(&translate.TranReq{
			From:  p.fromLang,
			To:    p.toLang,
			Paras: reqCont,
		}, len(contents)-len(reqCont))
	}

	p.wg.Wait()

	return contents
}

// WriteChanges 将处理后的内容写回 DOCX 文件
func (p *DocxProcessor) WriteChanges(translateContents []translate.Paragraph) error {
	// 1. 打开 docx 文件
	reader, err := zip.OpenReader(p.outputPath)
	if err != nil {
		return fmt.Errorf("failed to open docx file: %v", err)
	}
	defer reader.Close()

	// 2. 创建临时输出文件
	tmpPath := p.outputPath + ".tmp"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	writer := zip.NewWriter(tmpFile)

	paraIdx := 0 // 翻译段落索引

	// 3. 遍历 zip 文件条目
	for _, f := range reader.File {
		w, err := writer.Create(f.Name)
		if err != nil {
			return fmt.Errorf("failed to create file in zip: %v", err)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in zip: %v", err)
		}

		if f.Name == "word/document.xml" {
			// 修改 document.xml
			decoder := xml.NewDecoder(rc)
			encoder := xml.NewEncoder(w)
			inParagraph := false
			textIdx := 0

			for {
				token, err := decoder.Token()
				if err != nil {
					if err == io.EOF {
						break
					}
					return fmt.Errorf("decode error: %v", err)
				}

				switch se := token.(type) {
				case xml.StartElement:
					if se.Name.Local == "p" {
						inParagraph = true
						textIdx = 0
					}
					if inParagraph && se.Name.Local == "t" {
						var text struct {
							Content string `xml:",chardata"`
						}
						if err := decoder.DecodeElement(&text, &se); err == nil {
							if p.langChecker != nil && p.langChecker.Check(text.Content) {
								// 替换为翻译后的内容
								if paraIdx < len(translateContents) {
									if textIdx < len(translateContents[paraIdx]) {
										text.Content = translateContents[paraIdx][textIdx]
									}
								}
								textIdx++
								// 写回
								if err := encoder.EncodeElement(text, se); err != nil {
									return fmt.Errorf("encode error: %v", err)
								}
								continue
							}
						}
					}
				case xml.EndElement:
					if se.Name.Local == "p" && inParagraph {
						paraIdx++
						inParagraph = false
					}
				}

				if err := encoder.EncodeToken(token); err != nil {
					return fmt.Errorf("encode token error: %v", err)
				}
			}
			encoder.Flush()
		} else {
			// 其他文件直接复制
			_, err = io.Copy(w, rc)
			if err != nil {
				return fmt.Errorf("copy file error: %v", err)
			}
		}

		rc.Close()
	}

	writer.Close()
	tmpFile.Close()

	// 4. 替换原文件
	if err := os.Rename(tmpPath, p.outputPath); err != nil {
		return fmt.Errorf("failed to replace file: %v", err)
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
	contents, err := p.ExtractText()
	if err != nil {
		return err
	}
	log.Printf("extracted %d paragraphs", len(contents))
	// 3. 处理文本
	processedContent := p.ProcessText(contents)

	// 4. 写回修改
	return p.WriteChanges(processedContent)
}
