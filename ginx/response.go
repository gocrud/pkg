package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gocrud/pkg/errorx"
)

func FailParam(ctx *gin.Context) {
	err := errorx.E(errorx.ErrParam, "请求参数格式错误")
	_ = ctx.Error(err)
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
