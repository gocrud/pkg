package seatax

import (
	"fmt"

	"github.com/gocrud/pkg/errorx"
)

// newBizErr 构造携带指定错误码与用户提示的业务错误,cause 可选(仅第一个生效)。
func newBizErr(code, msg string, cause ...error) error {
	c := errorx.Define(code, msg)
	if len(cause) > 0 && cause[0] != nil {
		return c.Wrap(cause[0])
	}
	return c
}

// toError 将 panic 值规整为 error。
func toError(v any) error {
	if e, ok := v.(error); ok {
		return e
	}
	return fmt.Errorf("%v", v)
}

// keepBizError 保留业务错误的错误码与提示语,仅附加 seata 二阶段失败作为
// cause,保证 grpcx / ginx / microx 能按业务错误码继续渲染,不丢失上下文。
func keepBizError(err error, cause error) error {
	if biz, ok := errorx.ErrorOf(err); ok {
		return newBizErr(biz.CodeStr(), biz.Msg(), cause)
	}
	return newBizErr(CodeInternal, "分布式事务处理失败", cause)
}

// firstErr 在业务错误与 panic 中取先发生的那个。
func firstErr(panicValue any, bizErr error) error {
	if panicValue != nil {
		return toError(panicValue)
	}
	return bizErr
}
