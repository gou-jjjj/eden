package eden

//import (
//	"fmt"
//	"os"
//	"strings"
//
//	"github.com/gou-jjjj/eden/translate"
//	"github.com/gou-jjjj/unioffice/document"
//)
//
//func main() {
//	// 1. 打开文档
//	doc, err := document.Open("/Users/zyb/go/src/github.com/gou-jjjj/eden/file_examples/dxusercu_374966c9c16afa528c7cbed0ad763b12.docx")
//	if err != nil {
//		fmt.Println(err)
//	}
//	defer doc.Close()
//
//	var allParagraphs []translate.Paragraph
//
//	// 2. 提取普通段落文本
//	for _, para := range doc.Paragraphs() {
//		var texts []string
//		for _, run := range para.Runs() {
//			text := run.Text()
//			if text != "" {
//				texts = append(texts, text)
//			}
//		}
//		if len(texts) > 0 {
//			allParagraphs = append(allParagraphs, texts)
//		}
//	}
//
//	// 3. 提取表格中的文本
//	tableParagraphs := extractTables(doc)
//	allParagraphs = append(allParagraphs, tableParagraphs...)
//
//	// 4. 保存提取的内容
//	content := strings.Builder{}
//	for i, paragraph := range allParagraphs {
//		content.WriteString(fmt.Sprintf("段落 %d: ", i))
//		content.WriteString(strings.Join(paragraph, " | "))
//		content.WriteString("\n")
//	}
//
//	_ = os.WriteFile("./extracted.txt", []byte(content.String()), 0644)
//	fmt.Printf("总共提取了 %d 个文本段落（包括表格内容）", len(allParagraphs))
//}
//
//// 提取表格内容
//func extractTables(doc *document.Document) []translate.Paragraph {
//	var tableParagraphs []translate.Paragraph
//
//	for tableIndex, table := range doc.Tables() {
//		fmt.Printf("发现表格 %d，包含 %d 行", tableIndex, len(table.Rows()))
//
//		for rowIndex, row := range table.Rows() {
//			for cellIndex, cell := range row.Cells() {
//				// 提取单元格中的段落
//				var cellTexts []string
//				for paraIndex, para := range cell.Paragraphs() {
//					var runs []string
//					for runIndex, run := range para.Runs() {
//						text := run.Text()
//						if text != "" {
//							runs = append(runs, text)
//							fmt.Printf("表格[%d]行[%d]列[%d]段落[%d]运行[%d]: %s",
//								tableIndex, rowIndex, cellIndex, paraIndex, runIndex, text)
//						}
//					}
//					if len(runs) > 0 {
//						cellTexts = append(cellTexts, strings.Join(runs, ""))
//					}
//				}
//
//				if len(cellTexts) > 0 {
//					// 为每个单元格创建一个段落条目，包含位置信息
//					location := fmt.Sprintf("[表格%d-行%d-列%d]", tableIndex, rowIndex, cellIndex)
//					tableParagraphs = append(tableParagraphs, append([]string{location}, cellTexts...))
//				}
//			}
//		}
//	}
//
//	return tableParagraphs
//}
//
//// 更高级的表格提取（保持表格结构）
//func extractTablesWithStructure(doc *document.Document) []translate.Paragraph {
//	var structuredParagraphs []translate.Paragraph
//
//	for tableIndex, table := range doc.Tables() {
//		// 表格开始标记
//		structuredParagraphs = append(structuredParagraphs, []string{
//			fmt.Sprintf("=== 表格 %d 开始 ===", tableIndex),
//		})
//
//		// 提取每一行
//		for rowIndex, row := range table.Rows() {
//			var rowTexts []string
//			rowTexts = append(rowTexts, fmt.Sprintf("行%d:", rowIndex))
//
//			for cellIndex, cell := range row.Cells() {
//				var cellContent []string
//				for _, para := range cell.Paragraphs() {
//					for _, run := range para.Runs() {
//						if text := run.Text(); text != "" {
//							cellContent = append(cellContent, text)
//						}
//					}
//				}
//
//				cellText := strings.Join(cellContent, " ")
//				rowTexts = append(rowTexts, fmt.Sprintf("列%d:%s", cellIndex, cellText))
//			}
//
//			structuredParagraphs = append(structuredParagraphs, rowTexts)
//		}
//
//		// 表格结束标记
//		structuredParagraphs = append(structuredParagraphs, []string{
//			fmt.Sprintf("=== 表格 %d 结束 ===", tableIndex),
//		})
//	}
//
//	return structuredParagraphs
//}
//
//// 完整的文档处理示例
//func TestProcessDocxWithTables() {
//	// 打开文档
//	doc, err := document.Open("/Users/zyb/go/src/github.com/gou-jjjj/eden/file_examples/dxusercu_374966c9c16afa528c7cbed0ad763b12.docx")
//	if err != nil {
//		panic(err)
//
//	}
//	defer doc.Close()
//
//	fmt.Printf("文档信息: %d 个段落, %d 个表格", len(doc.Paragraphs()), len(doc.Tables()))
//
//	// 提取所有内容（段落 + 表格）
//	allContent := extractAllContent(doc)
//
//	// 保存原始内容
//	saveContentToFile(allContent, "./before_translation.txt")
//
//	// 这里可以添加翻译逻辑
//	// translatedContent := translateContent(allContent)
//
//	// 写回翻译后的内容（包括表格）
//	// if err := writeTranslatedContent(doc, translatedContent); err != nil {
//	//     t.Fatal(err)
//	// }
//
//	// 保存文档
//	if err := doc.SaveToFile("./processed_with_tables.docx"); err != nil {
//	}
//}
//
//// 提取所有内容（段落和表格）
//func extractAllContent(doc *document.Document) []translate.Paragraph {
//	var allContent []translate.Paragraph
//
//	// 提取普通段落
//	for i, para := range doc.Paragraphs() {
//		var texts []string
//		for _, run := range para.Runs() {
//			if text := run.Text(); text != "" {
//				texts = append(texts, text)
//			}
//		}
//		if len(texts) > 0 {
//			// 添加位置信息
//			location := fmt.Sprintf("[段落%d]", i)
//			allContent = append(allContent, append([]string{location}, texts...))
//		}
//	}
//
//	// 提取表格内容
//	for tableIndex, table := range doc.Tables() {
//		for rowIndex, row := range table.Rows() {
//			for cellIndex, cell := range row.Cells() {
//				for paraIndex, para := range cell.Paragraphs() {
//					var texts []string
//					for _, run := range para.Runs() {
//						if text := run.Text(); text != "" {
//							texts = append(texts, text)
//						}
//					}
//					if len(texts) > 0 {
//						location := fmt.Sprintf("[表格%d-行%d-列%d-段落%d]", tableIndex, rowIndex, cellIndex, paraIndex)
//						allContent = append(allContent, append([]string{location}, texts...))
//					}
//				}
//			}
//		}
//	}
//
//	return allContent
//}
//
//// 保存内容到文件
//func saveContentToFile(content []translate.Paragraph, filename string) error {
//	var output strings.Builder
//	for i, para := range content {
//		output.WriteString(fmt.Sprintf("%d. %s\n", i+1, strings.Join(para, " | ")))
//	}
//	return os.WriteFile(filename, []byte(output.String()), 0644)
//}
