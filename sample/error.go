package sample

//go:generate errcodegen
const (
	// 内部错误
	ERROR = 500
	// 请求参数错误
	INVALID_PARAMS = 400

	// 鉴权相关失败
	ERROR_AUTH_CHECK_TOKEN_FAIL = 20001

	// 标签已存在
	ERROR_EXIST_TAG = 10001
)
