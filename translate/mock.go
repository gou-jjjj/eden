package translate

type MockTran struct {
}

func NewMockTran() *MockTran {
	return &MockTran{}
}

func (t MockTran) T(r *TranReq) (Paragraph, error) {
	return r.Paras, nil
}

func (t MockTran) Name() string {
	return "Mock"
}
