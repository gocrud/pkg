// Package ginx 提供基于 Gin 的统一响应与错误处理中间件。
//
// 导入路径为 github.com/gocrud/pkg/ginx,包名与目录一致:
//
//	import "github.com/gocrud/pkg/ginx"
//
// # 响应接口
//
//   - Ok(ctx, data) / Msg(ctx, msg):HTTP 200,code 为 SUCCESS。
//   - Fail(ctx, err) / FailParam(ctx):登记错误并 Abort,由 AutoErrorInterceptor 渲染。
//   - AutoErrorInterceptor(logger):把 ctx.Errors 的最后一条错误转换为 Result。
//
// # 错误映射
//
// 错误经 errorx.ErrorOf 沿 Unwrap 链识别:参数校验错误与业务错误返回 HTTP 200(业务
// code + 用户消息);code 为 ErrInternal(ERR_SYS)时返回 HTTP 500 并隐藏细节;其余
// 返回 HTTP 500 兑底提示。统一响应结构为 Result(code/msg/data/errors)。
//
// 详见本包 README。
package ginx
