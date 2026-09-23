package seatax

import (
	"context"

	"go-micro.dev/v6/client"
	"go-micro.dev/v6/metadata"
	"go-micro.dev/v6/server"

	"seata.apache.org/seata-go/v2/pkg/constant"
	"seata.apache.org/seata-go/v2/pkg/tm"
)

// microClientWrapper 在发起 RPC 前把当前 seata 上下文中的 XID 写入 outgoing
// metadata,供下游服务恢复全局事务上下文。
type microClientWrapper struct {
	client.Client
}

func (w *microClientWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	return w.Client.Call(WithXIDContext(ctx), req, rsp, opts...)
}

func (w *microClientWrapper) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Stream, error) {
	return w.Client.Stream(WithXIDContext(ctx), req, opts...)
}

func (w *microClientWrapper) Publish(ctx context.Context, msg client.Message, opts ...client.PublishOption) error {
	return w.Client.Publish(WithXIDContext(ctx), msg, opts...)
}

// MicroClientTransactionWrapper go-micro 客户端包装器:注入 XID 到 outgoing
// metadata。所有客户端调用统一走 WithXIDContext,业务侧无需感知。
func MicroClientTransactionWrapper() client.Wrapper {
	return func(c client.Client) client.Client {
		return &microClientWrapper{Client: c}
	}
}

// MicroServerTransactionWrapper go-micro 服务端包装器:从 incoming metadata
// 恢复 XID 并初始化 seata 上下文,handler 内的分支事务由此获得 XID 环境。
func MicroServerTransactionWrapper() server.HandlerWrapper {
	return func(fn server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			md, ok := metadata.FromContext(ctx)
			if !ok {
				return fn(ctx, req, rsp)
			}
			xid := md[constant.XidKey]
			if xid == "" {
				xid = md[constant.XidKeyLowercase]
			}
			if xid != "" {
				ctx = tm.InitSeataContext(ctx)
				tm.SetXID(ctx, xid)
			}
			return fn(ctx, req, rsp)
		}
	}
}

// WithXIDContext 把当前 seata 上下文中的 XID 写入 go-micro outgoing metadata,
// 返回新 context。无 XID 时原样返回。
func WithXIDContext(ctx context.Context) context.Context {
	if !tm.IsSeataContext(ctx) {
		return ctx
	}
	xid := tm.GetXID(ctx)
	if xid == "" {
		return ctx
	}
	return metadata.Set(ctx, constant.XidKey, xid)
}
