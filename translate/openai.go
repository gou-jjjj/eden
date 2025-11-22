package translate

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
)

type AiTran struct {
	retry int
	mIdx  int

	ctx    context.Context
	models []llms.Model
	elog   *slog.Logger
}

func NewAiTran(ctx context.Context, elog *slog.Logger, retry int, models ...llms.Model) *AiTran {
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
		t.elog.WarnContext(t.ctx, "模型请求失败", slog.Any("err", err))
		llm = t.nextModel()
		if llm != nil {
			goto TRAN
		}
		return "", err
	}

	return llmResp, nil
}

func (t *AiTran) translationFlow(req *TranReq) error {
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
		return err
	}
	req.TranslatedText = call
	return nil
}

func (t *AiTran) splitFlow(req *TranReq) error {
	const prompt = `
请将以下JSON数据中 output.segment 字段的内容翻译成中文，并严格保持原有的JSON结构不变。

**要求：**

1. 只翻译 output.segment 数组中的文本
2. 保持 input 部分完全不变
3. 保持 output.text 字段不变
4. 保持数组结构和顺序不变

**输入格式示例：**

{"input": {"text": "hello world!", "segment": ["hello ", "world!"]}, "output": {"text": "你好 世界！", "segment": []}}


你给我的数据应该是：

{"input": {"text": "hello world!", "segment": ["hello ", "world!"]}, "output": {"text": "你好 世界！", "segment": ["你好 ", "世界！"]}}

数据如下:

%s
`

	data, _ := json.Marshal(m{
		I: msg{
			T: strings.Join(req.Paras, ""),
			S: req.Paras,
		},
		O: msg{
			T: req.TranslatedText,
			S: make([]string, 0),
		},
	})

	content := fmt.Sprintf(prompt, data)
	call, err := t.call(content, llms.WithJSONMode(), llms.WithTemperature(0.5))
	if err != nil {
		return err
	}

	req.TranslatedJson = call
	return nil
}

func (t *AiTran) getResFlow(req *TranReq) ([]string, error) {
	const prompt = `
请将以下JSON字符串中 output.segment 数据解析出来,通过%s隔开。

我给你的数据是：

{"input": {"text": "hello world!", "segment": ["hello ", "world!"]}, "output": {"text": "你好 世界！", "segment": ["你好 ", "世界！"]}}

你给我的结果应该是：

你好%s世界！

数据如下:

%s
`
	randStr := fmt.Sprintf("\n%d\n", rand.Int())
	content := fmt.Sprintf(prompt, randStr, randStr, req.TranslatedJson)
	call, err := t.call(content, llms.WithTemperature(0.1))
	if err != nil {
		return nil, err
	}

	return strings.Split(call, randStr), nil
}

func (t *AiTran) addLog(req *TranReq, res []string) {
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
		s.WriteString("\n-----------------------------\n")
	}

	_ = os.WriteFile(
		fmt.Sprintf("error_resp_%d.log", time.Now().Unix()),
		[]byte(s.String()),
		0644,
	)
	t.elog.Warn(
		"翻译结果段落数与请求段落数不匹配，可能存在部分翻译丢失",
		slog.Int("req_para_count", len(req.Paras)),
		slog.Int("resp_para_count", len(res)),
	)
}

func (t *AiTran) T(req *TranReq) ([]string, error) {
	err := t.translationFlow(req)
	if err != nil {
		return nil, err
	}

	err = t.splitFlow(req)
	if err != nil {
		return nil, err
	}

	flow, err := t.getResFlow(req)
	if err != nil {
		return nil, err
	}

	return flow, nil
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

type (
	msg struct {
		T string   `json:"text"`
		S []string `json:"segment"`
	}

	m struct {
		I msg `json:"input"`
		O msg `json:"output"`
	}
)
