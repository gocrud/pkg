package seatax

import (
	"github.com/gin-gonic/gin"

	ginintegration "seata.apache.org/seata-go/v2/pkg/integration/gin"
)

// GinTransactionMiddleware 返回 gin 全局事务中间件。
//
// 它从 HTTP 请求头读取全局事务 XID(TX_XID,兼容小写 tx_xid),写入请求上下文,
// 使下游业务代码(AT/TCC/XA 分支注册、sql 拦截等)能感知当前全局事务;
// 缺少 XID 时以 400 中断请求。
//
// 注意:使用 gin >= 1.8.1 时,需将 engine.ContextWithFallback 置为 true。
func GinTransactionMiddleware() gin.HandlerFunc {
	return ginintegration.TransactionMiddleware()
}
