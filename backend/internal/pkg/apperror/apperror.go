package apperror

import "errors"

// Kind 表示错误的种类——是用户的问题还是服务器的问题
type Kind int

const (
	// KindBadRequest 表示请求参数有问题（用户的问题），应该返回 400
	KindBadRequest Kind = iota + 1

	// KindInternal 表示服务器内部出了问题（我们的问题），应该返回 500
	KindInternal
)

// AppError 是一个携带"错误种类"信息的自定义错误
// 这样 Controller 就能根据 Kind 决定返回 400 还是 500
type AppError struct {
	Kind    Kind
	Message string
	Err     error // 原始错误（如果有的话），方便调试
}

func (e *AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "unknown error"
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// —— 工厂函数：方便 Service 层快速创建错误 ——

func BadRequest(msg string) *AppError {
	return &AppError{Kind: KindBadRequest, Message: msg}
}

func Internal(msg string) *AppError {
	return &AppError{Kind: KindInternal, Message: msg}
}

func WrapInternal(err error) *AppError {
	return &AppError{Kind: KindInternal, Message: "服务器内部错误", Err: err}
}

// IsBadRequest 快速判断一个错误是不是"用户的问题"
// 使用 Go 标准库的 errors.As 沿着错误链自动查找 AppError
func IsBadRequest(err error) bool {
	var ae *AppError
	return errors.As(err, &ae) && ae.Kind == KindBadRequest
}
