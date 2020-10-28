package sample

//go:generate codemsg
const (
	// 需要登录
	CodeLoginRequired = 1000 + iota
	// 当前帐号于%s在另一台设备登录，如不是本人操作，建议进行帐号密码操作，以防密码信息泄漏风险！
	CodeAnotherDeviceLogin
	// 内部错误
	ERROR = 500
	// 请求参数错误
	INVALID_PARAMS = 400

	// 鉴权相关失败
	ERROR_AUTH_CHECK_TOKEN_FAIL = 20001

	// 标签已存在
	ERROR_EXIST_TAG = 10001
)
