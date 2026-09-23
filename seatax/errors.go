package seatax

import (
	"errors"
	"fmt"

	"github.com/gocrud/pkg/errorx"
)

// bizErrorSniffer 与 grpcx / httpx / microx 的约定一致,用于识别 errorx.BizError。
type bizErrorSniffer interface {
	CodeStr() string
	MsgStr() string
}

// toError 将 panic 值规整为 error。
func toError(v interface{}) error {
	if e, ok := v.(error); ok {
		return e
	}
	return fmt.Errorf("%v", v)
}

// keepBizError 保留业务错误的错误码与提示语,仅附加 seata 二阶段失败作为
// cause,保证 grpcx / httpx / microx 能按业务错误码继续渲染,不丢失上下文。
func keepBizError(err error, cause error) error {
	var biz bizErrorSniffer
	if errors.As(err, &biz) {
		return errorx.E(biz.CodeStr(), biz.MsgStr(), cause)
	}
	return errorx.E(CodeInternal, "分布式事务处理失败", cause)
}

// firstErr 在业务错误与 panic 中取先发生的那个。
func firstErr(panicValue interface{}, bizErr error) error {
	if panicValue != nil {
		return toError(panicValue)
	}
	return bizErr
}
