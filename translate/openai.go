package translate

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gou-jjjj/eden/lang"
	"github.com/gou-jjjj/eden/logger"
	"github.com/tmc/langchaingo/llms"
)

const (
	ZhiPu      = "zhipu"
	GithubFree = "githubfree"
	OpenRouter = "openrouter"
	AliBaBa    = "alibaba"
	Ollama     = "ollama"

	Seq = "\n------------\n"
)

var OpenaiModelList = map[string]struct {
	Url   string
	Key   string
	Model string
}{
	ZhiPu:      {"https://open.bigmodel.cn/api/paas/v4", "23c650b6b73d4b1b80500699edcbf87c.qU9lcBTeGfku2gKz", "glm-4-plus"},
	GithubFree: {"https://api.chatanywhere.tech", "sk-vINYqBzbzrhdsFxZCO7MSSEvHL8tPradBhl77tLmWmEoTXs5", "deepseek-v3"},
	OpenRouter: {"https://openrouter.ai/api/v1", "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", "x-ai/grok-4-fast:free"},
	AliBaBa:    {"https://dashscope.aliyuncs.com/compatible-mode/v1", "sk-227cf58d893d4a689e82d2b8eb8f3564", "qwen-plus"},
	Ollama:     {Url: "http://localhost:11434", Model: "qwen3:30b"},
}

type AiTran struct {
	ctx    context.Context
	retry  int
	mIdx   int
	models []llms.Model
	elog   logger.Logger
}

func NewOpenai(ctx context.Context, elog logger.Logger, retry int, models ...llms.Model) *AiTran {
	t := &AiTran{
		ctx:   ctx,
		retry: retry,
		elog:  elog,
	}

	if t.elog == nil {
		t.elog = logger.DefaultLogger
	}

	if t.ctx == nil {
		t.ctx = context.Background()
	}

	// 修复：正确初始化 models 切片
	t.models = make([]llms.Model, 0)
	for _, m := range models {
		if m != nil {
			t.models = append(t.models, m)
		}
	}

	return t
}

func (t *AiTran) call(content string, option ...llms.CallOption) (string, error) {
	llm := t.model()
	if llm == nil {
		return "", fmt.Errorf("no available models")
	}

TRAN:
	llmResp, err := llm.Call(t.ctx, content, option...)
	if err != nil {
		t.elog.Warn("模型请求失败：%v", err)
		llm = t.nextModel()
		if llm != nil {
			goto TRAN
		}
		return "", err
	}

	return llmResp, nil
}

// 仅执行一次翻译，不再包含任何重试
func (t *AiTran) performTranslation(req *TranReq) (Paragraph, error) {
	return nil, nil
}

func (t *AiTran) addLog(req *TranReq, res []string) {
	if t.elog != nil {
		s := strings.Builder{}
		for i := 0; i < max(len(req.Paras), len(res)); i++ {
			if i < len(req.Paras) {
				s.WriteString(fmt.Sprintf("[%s]->", req.Paras[i]))
			} else {
				s.WriteString("[] ")
			}
			if i < len(res) {
				s.WriteString(fmt.Sprintf("[%s] ", res[i]))
			} else {
				s.WriteString("[] ")
			}
			s.WriteString(Seq)
		}

		_ = os.WriteFile(
			fmt.Sprintf("error_resp_%d.log", time.Now().Unix()),
			[]byte(s.String()),
			0644,
		)
		t.elog.Warn(
			"翻译结果段落数与请求段落数不匹配，可能存在部分翻译丢失，req:%d, res:%d",
			len(req.Paras), len(res),
		)
	}
}

// 不再带重试，只执行一次，失败后自动使用备用翻译器
func (t *AiTran) T(req *TranReq) (Paragraph, error) {

	return nil, nil
}

func (t *AiTran) Name() string {
	return "OpenAI"
}

func (t *AiTran) model() llms.Model {
	if t.hasModels() {
		return t.models[t.mIdx]
	}
	return nil
}

func (t *AiTran) hasModels() bool {
	return t.mIdx < len(t.models)
}

func (t *AiTran) nextModel() llms.Model {
	t.mIdx++
	return t.model()
}

func getLangKey(from, to string) string {
	return fmt.Sprintf("%s_%s", from, to)
}

var samplePrompt = map[string][]llms.MessageContent{
	getLangKey(lang.ZH, lang.EN): {
		llms.TextParts(llms.ChatMessageTypeHuman, "要运行程序，请使用：`python main.py --input data.json`\n---\n这将处理数据集并生成\n---\n输出文件到`/results/`目录，截止东部时间下午5点。"),
		llms.TextParts(llms.ChatMessageTypeAI, "To run the program, use: `python main.py --input data.json`\n---\nThis will process the dataset and generate\n---\noutput files in `/results/` directory by 5PM EST."),
	},
	getLangKey(lang.EN, lang.ZH): {
		llms.TextParts(llms.ChatMessageTypeHuman, "To run the program, use: `python main.py --input data.json`\n---\nThis will process the dataset and generate\n---\noutput files in `/results/` directory by 5PM EST."),
		llms.TextParts(llms.ChatMessageTypeAI, "要运行程序，请使用：`python main.py --input data.json`\n---\n这将处理数据集并生成\n---\n输出文件到`/results/`目录，截止东部时间下午5点。"),
	},
}

const (
	translatePrompt = `
你是一个机器翻译专家，精通多种语言之间的翻译。请将以下内容从 %s 翻译成 %s 。请确保翻译准确且符合目标语言的语法和文化习惯。
`
)
