package microx

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gocrud/pkg/errorx"
	"github.com/gocrud/veri"
	microerrors "go-micro.dev/v6/errors"
)

// defaultServerID 是服务端生成结构化错误时使用的默认 Id,
// 与 go-micro 客户端内置的 "go.micro.client" 对称。
const defaultServerID = "go.micro.server"

// bizErrorSniffer 是识别 errorx.BizError 的接口,与 grpcx 保持一致。
type bizErrorSniffer interface {
	CodeStr() string
	MsgStr() string
}

// ToMicroError 将业务 handler 返回的错误翻译成 go-micro 结构化错误
// (*microerrors.Error)。识别顺序与 grpcx.ToGRPCError 对齐:
//  1. veri.ValidationErrors → 400 参数校验失败(Reason=ERR_PARAM,字段明细拼入 Detail)
//  2. errorx.BizError(非内部错误) → 按业务码归类(ERR_UNAUTH→401,其余→400,Reason=业务码)
//  3. 其余(内部错误、未知错误) → 500 系统繁忙(Reason=ERR_SYS)
func ToMicroError(err error) error {
	if err == nil {
		return nil
	}

	var valErrs *veri.ValidationErrors
	if errors.As(err, &valErrs) {
		return &microerrors.Error{
			Id:     defaultServerID,
			Code:   HTTPBadRequest,
			Detail: "参数校验失败: " + joinFieldErrors(valErrs),
			Reason: errorx.ErrParam,
		}
	}

	var biz bizErrorSniffer
	if errors.As(err, &biz) && biz.CodeStr() != errorx.ErrInternal {
		return &microerrors.Error{
			Id:     defaultServerID,
			Code:   httpCodeForBiz(biz.CodeStr()),
			Detail: biz.MsgStr(),
			Reason: biz.CodeStr(),
		}
	}

	return &microerrors.Error{
		Id:     defaultServerID,
		Code:   HTTPInternalError,
		Detail: "系统繁忙，请稍后再试",
		Reason: errorx.ErrInternal,
	}
}

// FromMicroError 将 go-micro 客户端调用返回的错误还原成 errorx.BizError。
// 返回值 ok 表示是否已成功还原;ok=false 时返回原始错误。
func FromMicroError(err error) (bool, error) {
	if err == nil {
		return false, nil
	}
	merr, ok := microerrors.As(err)
	if !ok || merr == nil {
		return false, err
	}
	if merr.Reason != "" {
		return true, errorx.E(merr.Reason, merr.Detail)
	}
	// 非本库服务端或 Reason 丢失时,按 HTTP 状态码兜底归类。
	switch merr.Code {
	case HTTPBadRequest:
		return true, errorx.E(errorx.ErrParam, merr.Detail)
	case HTTPUnauthorized:
		return true, errorx.E(errorx.ErrUnauthorized, merr.Detail)
	case HTTPInternalError:
		return true, errorx.E(errorx.ErrInternal, merr.Detail)
	default:
		return false, err
	}
}

// httpCodeForBiz 将 errorx 业务码归类到 HTTP 状态码:
// 身份未认证单独映射 401,参数错误与自定义业务错误统一 400。
func httpCodeForBiz(code string) int32 {
	if code == errorx.ErrUnauthorized {
		return HTTPUnauthorized
	}
	return HTTPBadRequest
}

func joinFieldErrors(valErrs *veri.ValidationErrors) string {
	var b strings.Builder
	for i, fieldErr := range valErrs.Errors {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(fmt.Sprintf("%s:%s", fieldErr.Field, fieldErr.Message))
	}
	return b.String()
}
