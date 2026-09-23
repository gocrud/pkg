package seatax

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gocrud/pkg/errorx"
	"seata.apache.org/seata-go/v2/pkg/constant"
	"seata.apache.org/seata-go/v2/pkg/tm"
)

// GinTransactionMiddleware HTTP(gin)事务中间件:从请求头(TX_XID / tx_xid)
// 恢复 XID 并初始化 seata 上下文,使 handler 内的分支事务获得 XID 环境。
//
// 默认严格模式:请求未携带 XID 时以 SEATA_XID_MISSING 业务错误中止
// (c.Error + c.Abort),由 httpx 的 AutoErrorInterceptor 统一渲染为 HTTP 200
// 的 Result 结构。确需放行无 XID 请求时,使用 WithAllowMissingXID()
// 宽松模式按普通请求处理:
//
//	seatax.GinTransactionMiddleware()
//	seatax.GinTransactionMiddleware(seatax.WithAllowMissingXID())
//
// 注意:gin >= 1.8.1 时如需通过 c.Value() 读取 seata 上下文,须将引擎的
// ContextWithFallback 置为 true(seata-go 官方要求)。
func GinTransactionMiddleware(opts ...func(*ginMiddlewareOptions)) gin.HandlerFunc {
	o := &ginMiddlewareOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return func(c *gin.Context) {
		xid := c.GetHeader(constant.XidKey)
		if xid == "" {
			xid = c.GetHeader(constant.XidKeyLowercase)
		}
		if xid == "" {
			if !o.allowMissingXID {
				err := errorx.E(CodeXIDMissing, "缺少全局事务 XID")
				_ = c.Error(err)
				c.Abort()
				return
			}
		} else {
			newCtx := tm.InitSeataContext(c.Request.Context())
			tm.SetXID(newCtx, xid)
			c.Request = c.Request.WithContext(newCtx)
		}
		c.Next()
	}
}

type ginMiddlewareOptions struct {
	allowMissingXID bool
}

// WithAllowMissingXID 开启宽松模式:请求未携带 XID 时按普通请求放行。
func WithAllowMissingXID() func(*ginMiddlewareOptions) {
	return func(o *ginMiddlewareOptions) {
		o.allowMissingXID = true
	}
}

// InjectXIDHeader 手动向 http.Request 注入 XID 头(官方约定头名 XIDHeaderKey),
// 用于在全局事务内发起 HTTP 调用(通常不需要,SDK 中间件在服务端自动恢复)。
func InjectXIDHeader(req *http.Request, xid string) {
	req.Header.Set(XIDHeaderKey, xid)
}

// XIDHeaderKey 官方 gin 中间件约定的 XID 请求头名称(constant.XidKey)。
const XIDHeaderKey = constant.XidKey
