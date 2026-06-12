package sioyun

import "fmt"

// APIError 网关返回的业务错误。
type APIError struct {
	HTTPStatus int    `json:"-"`
	Code       int    `json:"code"`
	Msg        string `json:"msg"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("sioyun: [%d] %s (http=%d)", e.Code, e.Msg, e.HTTPStatus)
}

// 预定义错误码。
const (
	ErrCodeSuccess           = 0
	ErrCodeBadRequest        = 400
	ErrCodeUnauthorized      = 401
	ErrCodeInsufficientFunds = 402
	ErrCodeForbidden         = 403
	ErrCodeNotFound          = 404
	ErrCodeRateLimited       = 429
	ErrCodeInternalError     = 500
)

// IsInsufficientFunds 判断是否短信额度不足。
func IsInsufficientFunds(err error) bool {
	if e, ok := err.(*APIError); ok {
		return e.Code == ErrCodeInsufficientFunds
	}
	return false
}

// IsRateLimited 判断是否频率超限。
func IsRateLimited(err error) bool {
	if e, ok := err.(*APIError); ok {
		return e.Code == ErrCodeRateLimited
	}
	return false
}
