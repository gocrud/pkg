package seatax

import (
	"context"

	seataTM "seata.apache.org/seata-go/v2/pkg/tm"
)

// InitSeataContext 初始化 seata 上下文(透传 SDK 实现)。
func InitSeataContext(ctx context.Context) context.Context {
	return seataTM.InitSeataContext(ctx)
}

// IsSeataContext 判断 ctx 是否已初始化 seata 上下文。
func IsSeataContext(ctx context.Context) bool {
	return seataTM.IsSeataContext(ctx)
}

// IsGlobalTx 判断当前 ctx 是否处于全局事务中(已绑定 XID)。
func IsGlobalTx(ctx context.Context) bool {
	return seataTM.IsGlobalTx(ctx)
}

// GetXID 获取当前上下文中的全局事务 XID,不存在时返回空字符串。
func GetXID(ctx context.Context) string {
	return seataTM.GetXID(ctx)
}

// SetXID 向 seata 上下文绑定全局事务 XID。ctx 须先经 InitSeataContext 初始化,
// 否则调用无效果(SDK 行为)。
func SetXID(ctx context.Context, xid string) {
	seataTM.SetXID(ctx, xid)
}
