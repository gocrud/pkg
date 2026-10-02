// Package pkg 是 github.com/gocrud/pkg 的根包,承载整个组件库的顶层文档。
//
// # 组件
//
// 各功能位于独立子包,按需导入:
//
//	github.com/gocrud/pkg/errorx  声明式业务错误码:Define、CodeOf、ErrorOf、Wrap
//	github.com/gocrud/pkg/ginx    Gin 统一响应与错误中间件:Ok、Fail、FailParam、AutoErrorInterceptor
//	github.com/gocrud/pkg/grpcx   gRPC 校验与错误转换:UnaryServerValidationInterceptor、ToGRPCError、FromGRPCError
//	github.com/gocrud/pkg/microx  go-micro v6 校验与错误转换:ValidationHandlerWrapper、ErrorHandlerWrapper、ToMicroError、FromMicroError
//	github.com/gocrud/pkg/logx    zerolog 初始化与多目标输出:Config、NewInstance、AddLog
//	github.com/gocrud/pkg/gormx   GORM 模型基类:BaseModel
//	github.com/gocrud/pkg/gormctx 事务连接在 context 中的载体:TxKey
//	github.com/gocrud/pkg/store   数据访问入口:GormDB、Redis、AddGorm、AddRedis
//	github.com/gocrud/pkg/uow     本地事务编排:UnitOfWork、AddUnitOfWork
//	github.com/gocrud/pkg/seatax  Seata 分布式事务:WithGlobalTx、GetSqlDb、GetGormDb、NewSeata、XID 传播
//
// # 快速开始
//
// 业务错误用 errorx 声明,协议出口统一渲染:
//
//	var ErrUserNotFound = errorx.Define("USER_NOT_FOUND", "user {{.id}} not found")
//
//	return ErrUserNotFound.Str("id", id).Wrap(err) // 附加命名参数与 cause
//
// # 错误处理约定
//
//   - 业务错误用 errorx.Define 声明错误码,链式 setter(Str/Int/Int64/Any)附加命名参数,
//     Wrap 附加底层 cause。
//   - 协议出口(ginx / grpcx / microx)通过 errorx.ErrorOf 沿 Unwrap 链识别业务错误;
//     code 为 ErrInternal(ERR_SYS)时隐藏细节,仅返回兑底提示。
//   - 协议层错误码由各出口包自行定义:ginx.ErrOK/ErrParam/ErrInternal/ErrForbidden、
//     grpcx.ErrParam/ErrInternal、microx.ErrParam/ErrUnauthorized/ErrInternal。
//
// 完整使用说明与示例见根目录 README 与各子包的 README。
package pkg
