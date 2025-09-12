package translate

import (
	"encoding/json"
	"fmt"
	"log"
)

type MockTran struct {
}

func NewMockTran() *MockTran {
	return &MockTran{}
}

func (t MockTran) T(r *TranReq) ([]Paragraph, error) {
	for i := range r.Paras {
		for i2 := range r.Paras[i] {
			r.Paras[i][i2] = fmt.Sprintf("%d_%d", i, i2)
		}
	}

	req, _ := json.Marshal(r)
	log.Println(string(req))
	return r.Paras, nil
}
