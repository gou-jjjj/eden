package translate

type Paragraph []string

type TranReq struct {
	From  string    `json:"from"`
	To    string    `json:"to"`
	Paras Paragraph `json:"paras"`
}

type Translate interface {
	T(*TranReq) (Paragraph, error)
	Name() string
}

const (
	BaiDu = "baidu"
)

const (
	OpenAI = "openai"
)

var TranslateSet = map[string]Translate{
	//BaiDu: NewBaidu("http://api.fanyi.baidu.com/api/trans/vip/translate", "20220422001184836", "kRMl9t9LwAn7EFCLibz0"),
}
