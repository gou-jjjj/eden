package translate

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/gou-jjjj/eden/lang"
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
	elog   *slog.Logger
}

func NewOpenai(ctx context.Context, elog *slog.Logger, retry int, models ...llms.Model) *AiTran {
	t := &AiTran{
		ctx:   ctx,
		retry: retry,
		elog:  elog,
	}

	if t.elog == nil {
		t.elog = slog.Default()
	}

	if t.ctx == nil {
		t.ctx = context.Background()
	}

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
		t.elog.WarnContext(t.ctx, "模型请求失败：%v", err.Error())
		llm = t.nextModel()
		if llm != nil {
			goto TRAN
		}
		return "", err
	}

	return llmResp, nil
}

// 仅执行一次翻译，不再包含任何重试
func (t *AiTran) translationFlow(req *TranReq) (string, error) {
	const prompt = `
你是一个机器翻译专家，精通多种语言之间的翻译。请将以下内容从%s翻译成%s。
1. 请保留所有额外的符号，如代码块中的标点符号、缩进和换行符。
2. 请确保翻译准确且符合目标语言的语法和文化习惯。
3. 请保持每个段落的结构与原文一致，不要合并或拆分段落。
4. 如果遇到专有名词或技术术语，请尽量保留其原始形式，确保其在翻译后仍然易于识别和理解。
5. 请避免在翻译中添加任何额外的解释或注释。
6. 请严格按照提供的段落顺序进行翻译，确保翻译结果的段落顺序与原文一致。
7. 如果某个段落无法翻译，请返回原始段落内容，而不是省略它。
下面是需要翻译的内容：

%s
`
	content := fmt.Sprintf(prompt, req.From, req.To, strings.Join(req.Paras, ""))
	call, err := t.call(content)
	if err != nil {
		return "", err
	}

	return call, nil
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

func (t *AiTran) T(req *TranReq) (string, error) {
	flow, err := t.translationFlow(req)
	if err != nil {
		return "", err
	}

	return flow, nil
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
