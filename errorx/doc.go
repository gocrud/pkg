// Package errorx 提供声明式、类型安全的业务错误码定义与消息渲染。
//
// # 核心设计
//
//   - Code[T] 是不可变的错误码定义,同时实现 error,无参数错误可直接 return ErrX。
//   - 链式构建:ErrX.Str("id", id).Int("age", 18).Wrap(err),中间态 *Builder 也实现 error。
//   - 模板:错误携带默认模板与命名参数,渲染时按 {{.name}} 占位填充。
//   - 堆栈:Stack() 显式按需捕获完整调用栈(排除 Go 语言层面与 errorx 内部帧)。
//
// # 快速开始
//
//	var ErrUserNotFound = errorx.Define("USER_NOT_FOUND", "user {{.id}} not found")
//
//	return ErrUserNotFound.Str("id", id).Wrap(err)
//	code, ok := errorx.CodeOf(err) // "USER_NOT_FOUND"
//	e, ok := errorx.ErrorOf(err)   // e.Msg()、e.CodeStr()、e.StackStr()
//
// # 常用入口
//
//   - 定义:Define(code, msg),code 支持字符串与常见整数。
//   - 链式:Str / Int / Int64 / Any 设置命名参数,Wrap 附加 cause,Stack 捕获堆栈。
//   - 判定:errors.Is(err, ErrX)、CodeOf(err)、ErrorOf(err)(均沿 Unwrap 链查找)。
//   - 协议层通用码:ErrOK(SUCCESS)、ErrParam(ERR_PARAM)、ErrUnauthorized(ERR_UNAUTH)、
//     ErrInternal(ERR_SYS)。
//
// %v 渲染 "[CODE] 消息",%+v 追加箭头树堆栈与 caused by 链。详见本包 README。
package errorx
