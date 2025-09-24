package translate

import (
	"testing"
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
					From: "English",
					To:   "Chinese",
					Paras: []Paragraph{
						{"we are happy."},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			s := OpenaiModelList[OpenRouter]
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
