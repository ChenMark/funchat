package errors

import "net/http"

// AppError 统一错误结构
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *AppError) Error() string {
	return e.Message
}

// 通用错误码定义
var (
	// 系统错误
	ErrInternal     = &AppError{Code: 50000, Message: "服务器内部错误"}
	ErrBadRequest   = &AppError{Code: 40000, Message: "请求参数错误"}
	ErrUnauthorized = &AppError{Code: 40100, Message: "未授权，请先登录"}
	ErrForbidden    = &AppError{Code: 40300, Message: "无权限访问"}

	// 验证码错误
	ErrCodeExpired     = &AppError{Code: 40010, Message: "验证码已过期"}
	ErrCodeInvalid     = &AppError{Code: 40011, Message: "验证码错误"}
	ErrCodeTooFrequent = &AppError{Code: 40012, Message: "发送过于频繁，请稍后再试"}
	ErrCodeDailyLimit  = &AppError{Code: 40013, Message: "今日发送次数已达上限"}

	// 用户错误
	ErrPhoneInvalid   = &AppError{Code: 40020, Message: "请输入正确的手机号"}
	ErrUserNotFound   = &AppError{Code: 40401, Message: "用户不存在"}
	ErrUserExist      = &AppError{Code: 40901, Message: "用户已存在"}
	ErrPhoneBound     = &AppError{Code: 40902, Message: "该手机号已被其他账号绑定"}

	// 好友错误
	ErrAlreadyFriend      = &AppError{Code: 40910, Message: "已经是好友了"}
	ErrRequestDuplicate   = &AppError{Code: 40911, Message: "已发送过好友申请"}
	ErrRequestNotFound    = &AppError{Code: 40410, Message: "好友申请不存在"}
	ErrCannotAddSelf      = &AppError{Code: 40030, Message: "不能添加自己为好友"}
	ErrNotFriend          = &AppError{Code: 40310, Message: "不是好友关系"}
)

// NewAppError 创建自定义错误
func NewAppError(code int, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// GetHTTPStatus 根据业务错误码返回 HTTP 状态码
func (e *AppError) GetHTTPStatus() int {
	switch {
	case e.Code >= 50000:
		return http.StatusInternalServerError
	case e.Code >= 40400:
		return http.StatusNotFound
	case e.Code >= 40300:
		return http.StatusForbidden
	case e.Code >= 40100:
		return http.StatusUnauthorized
	case e.Code >= 40900:
		return http.StatusConflict
	default:
		return http.StatusBadRequest
	}
}
