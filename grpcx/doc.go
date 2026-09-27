// Package grpcx 提供 gRPC Unary 请求校验与 errorx 错误的双向转换。
//
// # 服务端
//
//   - UnaryServerValidationInterceptor(skipMethods...):对实现 Validate() error 的
//     请求执行校验,失败返回 InvalidArgument 并写入 x-biz-code 等 trailer;
//     不转换 handler 返回的错误。
//   - ToGRPCError(ctx, err):把 handler 错误转换为 gRPC status。业务错误映射为
//     Aborted 并携带业务 code 与消息,其余(含 ERR_SYS)映射为 Internal。
//
// # 客户端
//
// FromGRPCError(err, trailer) 依据 status 与 trailer 还原 errorx 业务错误,返回
// (converted bool, err error);调用方需用 grpc.Trailer(&trailer) 取回 trailer。
//
// Trailer 键:HeaderBizCode(x-biz-code)、HeaderBizReason(x-biz-reason)、
// HeaderBizDetail(x-biz-detail)。详见本包 README。
package grpcx
