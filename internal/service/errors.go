package service

import "errors"

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("not found")
)

// ValidationError 表示带有面向前端提示信息的参数校验错误。
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func (e *ValidationError) Unwrap() error {
	return ErrInvalidInput
}

func newValidationError(msg string) error {
	return &ValidationError{Message: msg}
}
