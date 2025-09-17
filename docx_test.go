package main

import (
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"html"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/beevik/etree"
	"github.com/gou-jjjj/eden/prompt"
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
		WithInput("/Users/zyb/go/src/github.com/gou-jjjj/eden/file_examples/Docx4j_GettingStarted.docx"),
		WithOutput("./out"),
		WithProcessFunc(translate.NewOpenai()),
		WithMaxWords(10000),
		WithMaxGo(3))

	err := pr.Process()
	if err != nil {
		t.Error(err)
	}
}

func TestName(t *testing.T) {
	// ... (打开 zip，读取 document.xml 数据)
	reader, err := zip.OpenReader("/Users/zyb/go/src/github.com/gou-jjjj/eden/file_examples/Docx4j_GettingStarted.docx")
	if err != nil {
		t.Log(err)
		return
	}
	defer reader.Close()

	builder := &strings.Builder{}

	for _, file := range reader.File {
		if file.Name == "word/document.xml" {
			rc, err := file.Open()
			if err != nil {
				t.Log(err)
				return
			}

			doc := etree.NewDocument()
			_, err = doc.ReadFrom(rc)
			if err != nil {
				t.Log(err)
				return
			}

			eles := doc.FindElements(".//w:t")
			for _, ele := range eles {
				builder.WriteString(ele.Text() + "\n")
			}
		}
	}

	os.WriteFile("./out.txt", []byte(builder.String()), 0644)
}

func TestName1(t *testing.T) {
	reader, err := zip.OpenReader("./file_examples/zlobinski2011.docx")
	if err != nil {
		t.Log(err)
		return
	}
	defer reader.Close()

	builder := &strings.Builder{}

	for _, file := range reader.File {
		if file.Name == "word/document.xml" {
			rc, err := file.Open()
			if err != nil {
				t.Log(err)
				return
			}
			defer rc.Close()

			decoder := xml.NewDecoder(rc)
			for {
				token, err := decoder.Token()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Log(err)
					break
				}

				switch se := token.(type) {
				case xml.StartElement:
					if se.Name.Local == "t" {
						// 读取文本内容
						var text struct {
							Content string `xml:",chardata"`
						}
						if err := decoder.DecodeElement(&text, &se); err != nil {
							t.Log(err)
							continue
						}
						builder.WriteString(text.Content + "\n")
					}
				}
			}
		}
	}

	os.WriteFile("./out.txt", []byte(builder.String()), 0644)
}

func TestName3(t *testing.T) {
	reader, err := zip.OpenReader("./file_examples/zlobinski2011.docx")
	if err != nil {
		t.Log(err)
		return
	}
	defer reader.Close()

	builder := &strings.Builder{}

	for _, file := range reader.File {
		if file.Name == "word/document.xml" {
			rc, err := file.Open()
			if err != nil {
				t.Log(err)
				return
			}
			defer rc.Close()

			// 读取整个 XML 内容
			data, err := io.ReadAll(rc)
			if err != nil {
				t.Log(err)
				return
			}

			// 使用正则表达式提取所有 w:t 元素的内容
			// 这种方法不依赖 XML 解析器，更简单直接
			re := regexp.MustCompile(`<w:t[^>]*>(.*?)</w:t>`)
			matches := re.FindAllStringSubmatch(string(data), -1)

			for _, match := range matches {
				if len(match) > 1 {
					// 处理 XML 转义字符
					text := html.UnescapeString(match[1])
					builder.WriteString(text + "\n")
				}
			}
		}
	}

	os.WriteFile("./out3.txt", []byte(builder.String()), 0644)

}

func TestName5(t *testing.T) {
	var data [][]string
	reader, err := zip.OpenReader("/Users/zyb/go/src/github.com/gou-jjjj/eden/file_examples/Docx4j_GettingStarted.docx")
	if err != nil {
		t.Log(err)
		return
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.Name == "word/document.xml" {
			rc, err := file.Open()
			if err != nil {
				t.Log(err)
				return
			}
			defer rc.Close()

			decoder := xml.NewDecoder(rc)
			inParagraph := false
			var row []string
			for {
				token, err := decoder.Token()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Log(err)
					break
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
							row = append(row, text.Content)
						}
					}
				case xml.EndElement:
					if se.Name.Local == "p" && inParagraph {
						// 段落结束，写入内容并换行
						data = append(data, row)
						row = make([]string, 0)
						inParagraph = false
					}
				}
			}
		}
	}

	marshal, _ := json.Marshal(data)
	os.WriteFile("./out4.txt", marshal, 0644)
}

func TestName11(t *testing.T) {
	translatePrompt := prompt.TranslatePrompt("english", "chinese")
	t.Log(translatePrompt)
}
