package errorx

import (
	"errors"
	"fmt"
	"runtime"
)

type BizErrorBehavior interface {
	CodeStr() string
	MsgStr() string
}

type BizError struct {
	Code    string
	UserMsg string
	cause   error
	stack   string
}

func (err BizError) Error() string    { return fmt.Sprintf("[%s]%s", err.Code, err.UserMsg) }
func (err BizError) Unwrap() error    { return err.cause }
func (err BizError) CodeStr() string  { return err.Code }
func (err BizError) MsgStr() string   { return err.UserMsg }
func (err BizError) StackStr() string { return err.stack }

func E(code, userMsg string, cause ...error) error {
	var actualCause error
	if len(cause) > 0 {
		actualCause = cause[0]
	}
	return BizError{Code: code, UserMsg: userMsg, cause: actualCause}
}

func Wrap(err error, defaultUserMsg string) error {
	if err == nil {
		return nil
	}
	var target BizError
	if errors.As(err, &target) {
		return err
	}
	var stack string
	if _, file, line, ok := runtime.Caller(1); ok {
		shortFile := file
		for index := len(file) - 1; index > 0; index-- {
			if file[index] == '/' {
				shortFile = file[index+1:]
				break
			}
		}
		stack = fmt.Sprintf("%s:%d", shortFile, line)
	}
	return BizError{Code: ErrInternal, UserMsg: defaultUserMsg, cause: err, stack: stack}
}
