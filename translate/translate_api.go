package translate

type Content struct {
	Id   int    `json:"id"`
	Data string `json:"data"`
}

type TranReq struct {
	From string    `json:"from"`
	To   string    `json:"to"`
	Data []Content `json:"data"`
}

type Translate interface {
	T(*TranReq) ([]Content, error)
}

const (
	BaiDu = "baidu"
)

const (
	OpenAI = "openai"
)

var TranslateSet = map[string]Translate{
	//BaiDu: NewBaidu("http://api.fanyi.baidu.com/api/trans/vip/translate", "20220422001184836", "kRMl9t9LwAn7EFCLibz0"),

	//ai
	OpenAI: NewOpenai(),
}
