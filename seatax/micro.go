package seatax

import (
	"context"

	"go-micro.dev/v6/client"
	"go-micro.dev/v6/metadata"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/server"

	"seata.apache.org/seata-go/v2/pkg/constant"
	"seata.apache.org/seata-go/v2/pkg/tm"
)

// MicroTransactionHandlerWrapper 返回 go-micro 服务端 HandlerWrapper。
//
// 从 incoming metadata 中读取全局事务 XID(TX_XID,兼容小写 tx_xid)并写入上下文,
// 使业务 handler 处于全局事务上下文中。go-micro 的 grpc transport 会把 metadata
// 键统一小写后透传,因此两种写法都兼容。
func MicroTransactionHandlerWrapper() server.HandlerWrapper {
	return func(next server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			if md, ok := metadata.FromContext(ctx); ok {
				xid, _ := md.Get(constant.XidKey)
				if xid == "" {
					xid, _ = md.Get(constant.XidKeyLowercase)
				}
				if xid != "" {
					ctx = tm.InitSeataContext(ctx)
					tm.SetXID(ctx, xid)
				}
			}
			return next(ctx, req, rsp)
		}
	}
}

// MicroTransactionCallWrapper 返回 go-micro 客户端 CallWrapper。
//
// 当前上下文处于全局事务时,把 XID 写入 metadata 随请求传播给下游服务。
func MicroTransactionCallWrapper() client.CallWrapper {
	return func(next client.CallFunc) client.CallFunc {
		return func(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
			if tm.IsSeataContext(ctx) {
				ctx = metadata.Set(ctx, constant.XidKey, tm.GetXID(ctx))
			}
			return next(ctx, node, req, rsp, opts)
		}
	}
}
