package errorx

import (
	"fmt"
	"strconv"
)

// CodeValue 是错误码允许的底层类型:字符串或常见整数。
type CodeValue interface {
	~string | ~int | ~int32 | ~int64 | ~uint | ~uint32 | ~uint64
}

// Code 是不可变的错误码定义,同时实现 error 接口,无参数错误可直接 return。
//
//	var ErrUserNotFound = errorx.Define("USER_NOT_FOUND", "user {{.id}} not found")
type Code[T CodeValue] struct {
	// Code 是原始类型的错误码,保留类型供业务读取。
	Code T

	key      string
	template string
}

// Error 渲染单行消息:"[CODE] 模板消息"(不含堆栈)。
func (c *Code[T]) Error() string { return oneLine(c.key, c.template, nil) }

// Stack 捕获从应用入口到调用点的完整堆栈(排除 Go 语言层面与 errorx 内部帧),
// 返回可继续链式的 Builder。
func (c *Code[T]) Stack() *Builder {
	b := newBuilder(c)
	b.stack = captureStack()
	return b
}

// Str 设置字符串参数并返回链式 Builder。
func (c *Code[T]) Str(key, val string) *Builder { return newBuilder(c).Str(key, val) }

// Int 设置 int 参数并返回链式 Builder。
func (c *Code[T]) Int(key string, val int) *Builder { return newBuilder(c).Int(key, val) }

// Int64 设置 int64 参数并返回链式 Builder。
func (c *Code[T]) Int64(key string, val int64) *Builder { return newBuilder(c).Int64(key, val) }

// Any 设置任意类型参数并返回链式 Builder。
func (c *Code[T]) Any(key string, val any) *Builder { return newBuilder(c).Any(key, val) }

// Wrap 终结错误码并附加底层 cause,返回不可变的 *Error。
func (c *Code[T]) Wrap(err error) *Error { return newBuilder(c).Wrap(err) }

func (c *Code[T]) codeRef() any         { return c }
func (c *Code[T]) codeKey() string      { return c.key }
func (c *Code[T]) templateStr() string  { return c.template }
func (c *Code[T]) argList() []kv        { return nil }
func (c *Code[T]) stackFrames() []frame { return nil }
func (c *Code[T]) causeErr() error      { return nil }

// Define 定义错误码并设置默认模板。code 可以是字符串或整数,
// 模板支持 {{.name}} 命名参数。
func Define[T CodeValue](code T, msg string) *Code[T] {
	return &Code[T]{Code: code, key: canonical(code), template: msg}
}

// canonical 把错误码规范化为字符串:string 原样,数字 strconv 化。
func canonical[T CodeValue](code T) string {
	switch v := any(code).(type) {
	case string:
		return v
	case int:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	default:
		return fmt.Sprint(any(code))
	}
}

// 协议层通用错误码。与具体业务无关,供 ginx / grpcx / microx 等出口层
// 统一归类;业务自定义错误码请使用 Define。
const (
	// ErrOK 成功。
	ErrOK = "SUCCESS"
	// ErrInternal 内部错误。出口层据此隐藏细节,仅返回兑底提示。
	ErrInternal = "ERR_SYS"
	// ErrParam 参数错误。
	ErrParam = "ERR_PARAM"
	// ErrUnauthorized 未认证或凭证失效。
	ErrUnauthorized = "ERR_UNAUTH"
)
