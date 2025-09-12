package translate

import (
	"crypto/md5"
	"encoding/hex"
)

type Baidu struct {
	AppId  string
	AppKey string

	Domain string
}

func NewBaidu(domain, appid, appkey string) *Baidu {
	return &Baidu{
		AppId:  appid,
		AppKey: appkey,
		Domain: domain,
	}
}

func (b *Baidu) T(req *TranReq) ([]Paragraph, error) {
	return nil, nil
	//contents := make([]Content, len(req.Data))
	//
	//const (
	//	MaxWorkSize = 5000 // 每次翻译的最大字符数
	//)
	//
	//
	//// 生成 salt
	//rand.Seed(time.Now().UnixNano())
	//salt := rand.Intn(32768) + 32768
	//
	//// 生成 sign
	//signStr := fmt.Sprintf("%s%s%d%s", b.AppId, query, salt, b.AppKey)
	//sign := makeMd5(signStr)
	//
	//// 构建请求参数
	//data := url.Values{}
	//data.Set("q", query)
	//data.Set("from", fromLang)
	//data.Set("to", toLang)
	//data.Set("appid", b.AppId)
	//data.Set("salt", fmt.Sprintf("%d", salt))
	//data.Set("sign", sign)
	//
	//// 发送 POST 请求
	//resp, err := http.PostForm(b.Domain, data)
	//if err != nil {
	//	panic(err)
	//}
	//defer resp.Body.Close()
	//
	//body, _ := io.ReadAll(resp.Body)
	//
	//// 打印结果
	//var result map[string]interface{}
	//if err := json.Unmarshal(body, &result); err != nil {
	//	panic(err)
	//}
	//
	//output, _ := json.MarshalIndent(result, "", "    ")
	//return string(output), nil
}

func makeMd5(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
