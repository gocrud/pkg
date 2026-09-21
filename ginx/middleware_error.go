package httpx

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gocrud/pkg/errorx"
	"github.com/gocrud/veri"
	"github.com/rs/zerolog"
)

type stackTracer interface {
	CodeStr() string
	MsgStr() string
	StackStr() string
}

type unwrappable interface {
	Unwrap() error
}

type FieldErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type Result struct {
	Code   string             `json:"code"`
	Data   interface{}        `json:"data,omitempty"`
	Msg    string             `json:"msg"`
	Errors []FieldErrorDetail `json:"errors,omitempty"`
}

func AutoErrorInterceptor(logger zerolog.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()
		if ctx.Writer.Written() || len(ctx.Errors) == 0 {
			return
		}
		lastErr := ctx.Errors.Last().Err
		var valErrs *veri.ValidationErrors
		if errors.As(lastErr, &valErrs) {
			details := make([]FieldErrorDetail, 0, len(valErrs.Errors))
			for _, fieldErr := range valErrs.Errors {
				details = append(details, FieldErrorDetail{Field: fieldErr.Field, Message: fieldErr.Message})
			}
			logger.Debug().Int("errors_count", len(details)).Msg("validation error")
			ctx.AbortWithStatusJSON(http.StatusOK, Result{
				Code: errorx.ErrParam, Msg: "参数校验未通过", Errors: details,
			})
			return
		}
		if tracer, ok := lastErr.(stackTracer); ok {
			code := tracer.CodeStr()
			if code == errorx.ErrInternal {
				var realCause error
				if unwrapper, canUnwrap := lastErr.(unwrappable); canUnwrap {
					realCause = unwrapper.Unwrap()
				}
				logger.Error().Err(realCause).
					Str("caller", tracer.StackStr()).Str("biz_code", code).
					Msg("internal server error")
				ctx.AbortWithStatusJSON(http.StatusInternalServerError, Result{
					Code: code, Msg: "系统繁忙，请稍后再试",
				})
				return
			}
			ctx.AbortWithStatusJSON(http.StatusOK, Result{Code: code, Msg: tracer.MsgStr()})
			return
		}
		logger.Error().Err(lastErr).Msg("unhandled error intercepted")
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, Result{
			Code: errorx.ErrInternal, Msg: "服务异常",
		})
	}
}
