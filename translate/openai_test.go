package translate

import (
	"testing"

	"github.com/gou-jjjj/eden/lang"
)

func TestTranOpenai_T(t1 *testing.T) {
	type args struct {
		req *TranReq
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test1",
			args: args{
				req: &TranReq{
					From: lang.EN,
					To:   lang.ZH,
					Paras: Paragraph{
						"Hello, how are you?",
						"Today is a sunny day.",
						"OpenAI provides powerful AI models.",
						"Let's translate these sentences.",
						"Testing the translation functionality.",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			s := OpenaiModelList[AliBaBa]
			t := &TranOpenai{
				url:   s.Url,
				key:   s.Key,
				model: s.Model,
			}
			got, err := t.T(tt.args.req)
			if err != nil {
				t1.Errorf("TranOpenai.T() error = %v", err)
				return
			}
			t1.Log(got)
		})
	}
}
