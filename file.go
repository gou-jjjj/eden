package main

// DocxProcessor 定义 DOCX 文件处理接口
type Processor interface {
	// Copy 复制 DOCX 文件到指定目录
	Copy(inputPath, outDir string) (string, error)

	// ExtractText 从 DOCX 文件中提取文本内容
	ExtractText(filePath string, need func(string) bool) (Document, map[string]string, error)

	// ProcessText 处理文本内容
	ProcessText(textMap map[string]string, fromLang, toLang string, process func(string) string) map[string]string

	// WriteChanges 将处理后的内容写回 DOCX 文件
	WriteChanges(filePath string, doc Document, textMap map[string]string) error

	// Process 执行完整的 DOCX 处理流程
	Process(inputPath, outDir, fromLang, toLang string, process func(string) string) error
}

const (
	Docx = "docx"
)
