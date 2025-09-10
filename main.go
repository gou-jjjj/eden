package main

func main() {

}

//
//import (
//	"archive/zip"
//	"encoding/xml"
//	"fmt"
//	"io"
//	"os"
//	"unicode"
//)
//
//// Document 代表 docx 文件中的 XML 文档结构
//type Document struct {
//	XMLName xml.Name `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main document"`
//	Body    Body     `xml:"body"`
//}
//
//// Body 包含段落列表
//type Body struct {
//	Paragraphs []Paragraph `xml:"p"`
//}
//
//// Paragraph 包含文本运行列表
//type Paragraph struct {
//	Runs []Run `xml:"r"`
//}
//
//// Run 包含文本内容
//type Run struct {
//	Text []Text `xml:"t"`
//}
//
//// Text 包含实际文本值
//type Text struct {
//	Value string `xml:",chardata"`
//}
//
//// containsChinese 判断字符串是否包含汉字
//func containsChinese(s string) bool {
//	for _, r := range s {
//		if unicode.Is(unicode.Han, r) {
//			return true
//		}
//	}
//	return false
//}
//
//// readDocumentXML 从 docx 文件中读取 document.xml
//func readDocumentXML(reader *zip.Reader) (Document, error) {
//	var doc Document
//	for _, f := range reader.File {
//		if f.Name == "word/document.xml" {
//			rc, err := f.Open()
//			if err != nil {
//				return Document{}, fmt.Errorf("failed to open %s: %v", f.Name, err)
//			}
//			defer rc.Close()
//
//			data, err := io.ReadAll(rc)
//			if err != nil {
//				return Document{}, fmt.Errorf("failed to read %s: %v", f.Name, err)
//			}
//
//			if err := xml.Unmarshal(data, &doc); err != nil {
//				return Document{}, fmt.Errorf("failed to parse document.xml: %v", err)
//			}
//			return doc, nil
//		}
//	}
//	return Document{}, fmt.Errorf("word/document.xml not found in docx file")
//}
//
//// processDocument 修改文档中的文本内容
//func processDocument(doc Document, process func(string) string) (Document, error) {
//	for i := range doc.Body.Paragraphs {
//		for j := range doc.Body.Paragraphs[i].Runs {
//			for k := range doc.Body.Paragraphs[i].Runs[j].Text {
//				doc.Body.Paragraphs[i].Runs[j].Text[k].Value = process(doc.Body.Paragraphs[i].Runs[j].Text[k].Value)
//			}
//		}
//	}
//	return doc, nil
//}
//
//// writeDocumentXML 将修改后的文档写入新的 docx 文件
//func writeDocumentXML(reader *zip.Reader, outputPath string, doc Document) error {
//	outputFile, err := os.Create(outputPath)
//	if err != nil {
//		return fmt.Errorf("failed to create output docx file: %v", err)
//	}
//	defer outputFile.Close()
//
//	zipWriter := zip.NewWriter(outputFile)
//	defer zipWriter.Close()
//
//	// Marshal modified XML
//	documentXML, err := xml.MarshalIndent(doc, "", "  ")
//	if err != nil {
//		return fmt.Errorf("failed to marshal modified XML: %v", err)
//	}
//
//	// Process all files in the input docx
//	for _, f := range reader.File {
//		writer, err := zipWriter.Create(f.Name)
//		if err != nil {
//			return fmt.Errorf("failed to create zip entry %s: %v", f.Name, err)
//		}
//
//		if f.Name == "word/document.xml" {
//			// Write modified document.xml
//			_, err = writer.Write([]byte(xml.Header + string(documentXML)))
//			if err != nil {
//				return fmt.Errorf("failed to write modified document.xml: %v", err)
//			}
//		} else {
//			// Copy other files unchanged
//			rc, err := f.Open()
//			if err != nil {
//				return fmt.Errorf("failed to open %s: %v", f.Name, err)
//			}
//			_, err = io.Copy(writer, rc)
//			rc.Close()
//			if err != nil {
//				return fmt.Errorf("failed to copy %s: %v", f.Name, err)
//			}
//		}
//	}
//
//	return nil
//}
//
//// processDocx 处理 docx 文件：读取、修改、写入
//func processDocx(inputPath, outputPath string, process func(string) string) error {
//	// Open input docx file
//	reader, err := zip.OpenReader(inputPath)
//	if err != nil {
//		return fmt.Errorf("failed to open input docx file: %v", err)
//	}
//	defer reader.Close()
//
//	// Read document.xml
//	doc, err := readDocumentXML(reader)
//	if err != nil {
//		return err
//	}
//
//	// Process document
//	modifiedDoc, err := processDocument(doc, process)
//	if err != nil {
//		return err
//	}
//
//	// Write to new docx file
//	return writeDocumentXML(reader, outputPath, modifiedDoc)
//}
//
//// tran 模拟翻译函数（需替换为实际翻译 API 调用）
//func tran(text string) string {
//	// 这里应调用实际的翻译 API，例如之前的 Translate 函数
//	// 为示例，简单返回一个假翻译
//	return "Translated: " + text
//}
//
//func main() {
//	inputPath := "/Users/zyb/go/src/flow/docxxml/dxusercu_e43caac4e7a606e6f290e3718d67ce21.docx"
//	outputPath := "/Users/zyb/go/src/flow/docxxml/dxuse.docx"
//
//	// Define text processing function
//	processor := func(text string) string {
//		if !containsChinese(text) {
//			return text
//		}
//		s := tran(text)
//		if s != "" {
//			fmt.Println("原文:", text, "-> 翻译:", s)
//			return s
//		}
//		return text
//	}
//
//	err := processDocx(inputPath, outputPath, processor)
//	if err != nil {
//		fmt.Printf("Error: %v\n", err)
//		os.Exit(1)
//	}
//
//	fmt.Printf("New docx file created: %s\n", outputPath)
//}
