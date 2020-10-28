// Code generated by github.com/pwli0755/codemsg DO NOT EDIT
// Source: sample/error.go
// error_msg_gen.go is a generated file.

package sample

// messages get msg from const comment
var messages = map[int]string{

	CodeAnotherDeviceLogin:      "当前帐号于%s在另一台设备登录，如不是本人操作，建议进行帐号密码操作，以防密码信息泄漏风险！",
	CodeLoginRequired:           "需要登录",
	ERROR:                       "内部错误",
	ERROR_AUTH_CHECK_TOKEN_FAIL: "鉴权相关失败",
	ERROR_EXIST_TAG:             "标签已存在",
	INVALID_PARAMS:              "请求参数错误",
}

// GetMsg get code msg
func GetMsg(code int) string {
	if msg, ok := messages[code]; ok {
		return msg
	}
	return "UNKNOWN ERROR"
}
