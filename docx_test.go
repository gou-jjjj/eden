package eden

import (
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/gou-jjjj/eden/lang"
	"github.com/gou-jjjj/eden/logger"
	"github.com/gou-jjjj/eden/prompt"
	"github.com/gou-jjjj/eden/translate"
	"github.com/gou-jjjj/unioffice/document"
)

func debugDelete() {
	err := os.RemoveAll("./out")
	if err != nil {
		log.Fatal(err)
	}
}

func TestNewDocxProcessor(t *testing.T) {
	//debugDelete()
	newLogger, err := logger.NewLogger(true, "./out", "test_docx_processor")
	if err != nil {
		t.Log(err)
		return
	}
	newLogger.SetLevel(logger.INFO)
	//open := translate.NewOpenai(translate.OpenRouter)

	pr := NewDocxProcessor(
		WithInput("/Users/calvin/go/src/eden/file_examples/Docx4j_GettingStarted.docx"),
		WithOutput("./out"),
		WithLang(lang.ZH),
		//WithProcessFunc(translate.NewOpenaiWithLogger(translate.AliBaBa, newLogger, open)),
		WithProcessFunc(translate.NewMockTran()),
		WithMaxGo(10),
		WithLogger(newLogger),
		WithMaxToken(100))

	err = pr.Process()
	if err != nil {
		t.Error(err)
	}

	// 检查日志文件是否生成
	if pr.logger != nil {
		logPath := pr.logger.GetLogFilePath()
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			t.Errorf("日志文件未生成: %s", logPath)
		} else {
			t.Logf("日志文件已生成: %s", logPath)
		}
	}
}

func TestContent(t *testing.T) {
	const path = "/Users/calvin/go/src/eden/file_examples/Docx4j_GettingStarted.docx"
	doc, err := document.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()

	// 检查文档基本信息
	t.Logf("段落数量: %d", len(doc.Paragraphs()))
	t.Logf("表格数量: %d", len(doc.Tables()))

	// 检查段落详情
	for _, para := range doc.Paragraphs() {
		for _, run := range para.Runs() {
			fmt.Print(run.Text())
		}
		fmt.Println()
	}
}

func TestPrompt(t *testing.T) {
	translatePrompt := prompt.TranslatePrompt("english", "chinese")
	t.Logf("%v\n", translatePrompt)
}

func TestLogging(t *testing.T) {
	debugDelete()

	// 创建自定义日志记录器
	loggerInstance, err := logger.NewLogger(true, "./out", "test_logging")
	if err != nil {
		t.Fatalf("创建日志记录器失败: %v", err)
	}
	defer loggerInstance.Close()

	// 设置日志级别为DEBUG以查看详细信息
	loggerInstance.SetLevel(logger.DEBUG)

	pr := NewDocxProcessor(WithLang(lang.All, lang.EN),
		WithInput("C:\\Users\\Administrator\\go\\src\\eden\\file_examples\\dxusercu_e43caac4e7a606e6f290e3718d67ce21.docx"),
		WithOutput("./out"),
		WithProcessFunc(translate.NewMockTran()),
		WithMaxGo(1),
		WithLogger(loggerInstance))

	err = pr.Process()
	if err != nil {
		t.Error(err)
	}

	// 验证日志文件内容
	logPath := loggerInstance.GetLogFilePath()
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Errorf("读取日志文件失败: %v", err)
		return
	}

	logContent := string(content)

	// 检查关键日志信息
	expectedLogs := []string{
		"翻译任务开始",
		"文件加载成功",
		"文本提取完成",
		"开始翻译",
		"翻译完成",
		"翻译结果写回完成",
		"文件保存成功",
		"翻译统计",
		"翻译任务完成",
	}

	for _, expectedLog := range expectedLogs {
		if !strings.Contains(logContent, expectedLog) {
			t.Errorf("日志中缺少预期内容: %s", expectedLog)
		}
	}

	t.Logf("日志文件内容预览:\n%s", logContent[:min(500, len(logContent))])
}
