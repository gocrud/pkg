package ginx

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gocrud/pkg/errorx"
)

// errParamFormat 参数格式错误的哨兵(携带用户提示)。
var errParamFormat = errorx.Define(errorx.ErrParam, "请求参数格式错误")

func FailParam(ctx *gin.Context) {
	_ = ctx.Error(errParamFormat)
	ctx.Abort()
}

func Ok(ctx *gin.Context, data interface{}) {
	ctx.JSON(http.StatusOK, Result{Code: errorx.ErrOK, Msg: "success", Data: data})
}

func Msg(ctx *gin.Context, msg string) {
	ctx.JSON(http.StatusOK, Result{Code: errorx.ErrOK, Msg: msg})
}

func Fail(ctx *gin.Context, err error) {
	if err == nil {
		return
	}
	_ = ctx.Error(err)
	ctx.Abort()
}
