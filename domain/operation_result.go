package domain

func NewOperationResult(err error) *OperationResult {
	r := &OperationResult{}
	if err != nil {
		r.ErrMsg = err.Error()
	}
	return r
}
