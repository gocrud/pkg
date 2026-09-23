package seatax

import (
	"context"

	"google.golang.org/grpc"
	seataGrpc "seata.apache.org/seata-go/v2/pkg/integration/grpc"
)

// ServerTransactionInterceptor gRPC 服务端拦截器:从上游携带的 metadata
// 中恢复 XID 并初始化 seata 上下文,为服务内的分支事务提供 XID 环境。
// 配合错误处理拦截器使用:
//
//	grpc.NewServer(grpc.ChainUnaryInterceptor(
//		seatax.ServerTransactionInterceptor(),
//		grpcx.UnaryServerValidationInterceptor(),
//	))
func ServerTransactionInterceptor() grpc.UnaryServerInterceptor {
	return seataGrpc.ServerTransactionInterceptor
}

// ClientTransactionInterceptor gRPC 客户端拦截器:向调用目标的 metadata
// 注入当前 seata 上下文中的 XID。用于 RPC 链路上游侧。
//
//	conn, _ := grpc.Dial(target, grpc.WithUnaryInterceptor(seatax.ClientTransactionInterceptor()))
func ClientTransactionInterceptor() grpc.UnaryClientInterceptor {
	return seataGrpc.ClientTransactionInterceptor
}

// ClientTransactionStreamInterceptor gRPC 客户端流式拦截器:
// 在流调用发起时注入 XID。
func ClientTransactionStreamInterceptor() grpc.StreamClientInterceptor {
	return seataGrpc.ClientTransactionStreamInterceptor
}

// InjectXIDContext 手动向 gRPC outgoing context 注入当前 XID(通常不需要,
// 客户端拦截器已处理;用于自定义元数据处理场景)。
func InjectXIDContext(ctx context.Context) context.Context {
	if xid := GetXID(ctx); xid != "" {
		return context.WithValue(ctx, grpcXIDContextKey{}, xid)
	}
	return ctx
}

type grpcXIDContextKey struct{}
