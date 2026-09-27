// Package microx 提供 go-micro v6 的请求校验与 errorx 错误转换,与 grpcx 对齐。
//
// go-micro 结构化错误 *errors.Error 自带 Id/Code/Detail/Reason:业务码放入 Reason,
// Code 采用 HTTP 风格并由 server/grpc 映射为 gRPC status,因此无需 trailer。
//
//   - ValidationHandlerWrapper(skipEndpoints...):对实现 Validate() error 的请求执行校验。
//   - ErrorHandlerWrapper():把 handler 错误转换为 go-micro 结构化错误。
//   - ToMicroError(err):参数校验 → 400/ERR_PARAM;业务错误 → 401(ERR_UNAUTH)或 400;
//     其余 → 500/ERR_SYS(隐藏原因)。
//   - FromMicroError(err):还原 errorx 业务错误,返回 (converted bool, err error)。
//
// 详见本包 README。
package microx
