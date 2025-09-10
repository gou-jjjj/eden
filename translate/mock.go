package translate

import (
	"encoding/json"
	"log"
)

type MockTran struct {
}

func NewMockTran() *MockTran {
	return &MockTran{}
}

func (t MockTran) T(r *TranReq) ([]Content, error) {
	req, _ := json.Marshal(r)
	log.Println(string(req))
	return r.Data, nil
}
