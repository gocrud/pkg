package seatax

import (
	"google.golang.org/grpc"

	grpcintegration "seata.apache.org/seata-go/v2/pkg/integration/grpc"
)

// GrpcClientTransactionInterceptor 返回 gRPC 客户端拦截器。
//
// 当前上下文处于全局事务时,把 XID 注入 outgoing metadata 随请求传播给下游,
// 并在调用前后记录 RPC 耗时与错误。
func GrpcClientTransactionInterceptor() grpc.UnaryClientInterceptor {
	return grpcintegration.ClientTransactionInterceptor
}

// GrpcServerTransactionInterceptor 返回 gRPC 服务端拦截器。
//
// 从 incoming metadata 中取出 XID(TX_XID,兼容小写 tx_xid)并写入上下文,
// 使业务处理处于全局事务上下文中。
func GrpcServerTransactionInterceptor() grpc.UnaryServerInterceptor {
	return grpcintegration.ServerTransactionInterceptor
}

// GrpcClientStreamTransactionInterceptor 返回 gRPC 流式客户端拦截器,逻辑同 unary。
func GrpcClientStreamTransactionInterceptor() grpc.StreamClientInterceptor {
	return grpcintegration.ClientTransactionStreamInterceptor
}
