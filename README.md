# gocrud/pkg

Go 服务端公共组件库：业务错误、HTTP 响应、gRPC / go-micro 错误转换、GORM 数据访问与事务、结构化日志、Seata 分布式事务。模块路径 `github.com/gocrud/pkg`。

## 安装

Go 1.27+，以 [go.mod](go.mod) 为准。

```sh
go get github.com/gocrud/pkg
```

生产项目建议固定经过验证的 tag 或 commit。

## 包

| 导入路径 | 包名 | 说明 | 文档 |
| --- | --- | --- | --- |
| `/errorx` | `errorx` | 声明式业务错误码、链式构建、堆栈 | [errorx/README.md](errorx/README.md) |
| `/ginx` | `ginx` | Gin 统一响应与错误中间件 | [ginx/README.md](ginx/README.md) |
| `/grpcx` | `grpcx` | gRPC 校验与错误转换 | [grpcx/README.md](grpcx/README.md) |
| `/microx` | `microx` | go-micro v6 校验与错误转换 | [microx/README.md](microx/README.md) |
| `/logx` | `logx` | zerolog 初始化与输出 | [logx/README.md](logx/README.md) |
| `/seatax` | `seatax` | Seata 分布式事务与 XID 传播 | [seatax/README.md](seatax/README.md) |

## 通用约定

- 错误统一走 `errorx`：`Define(code, msg)` 声明错误码，`CodeOf` / `ErrorOf` 提取业务码与消息。
- 协议层错误码由各出口包自行定义（`ginx.ErrOK/ErrParam/ErrInternal/ErrForbidden`、`grpcx.ErrParam/ErrInternal`、`microx.ErrParam/ErrUnauthorized/ErrInternal`），`errorx` 只提供 `Define` 与渲染。
- 协议出口（ginx / grpcx / microx）通过 `errorx.ErrorOf` 沿 `Unwrap` 链识别业务错误并渲染，内部错误不向客户端暴露原因。
- 依赖注入：组件扩展遵循 [github.com/gocrud/kernel](https://github.com/gocrud/kernel) 的 `Extension` 约定（只注册、无返回值，内部用 `TryProvide` 保证幂等）。无参扩展直接传函数：`infra.AddStore`、`infra.AddUnitOfWork`、`seatax.AddSeata`；需要参数时返回 `kernel.Extension`：`logx.AddLog(cfg)`、`seatax.AddGormDb(driver, dsn)`。装配时用 `kernel.New().Extend(...)` 链式调用，`Build()` 一次性构造并就绪：

  ```go
  app, err := kernel.New().
      Extend(seatax.AddGormDb(seatax.XAPostgres, dsn)). // 注册 *gorm.DB（seata 代理连接）
      Extend(infra.AddStore).                           // 依赖 *gorm.DB
      Extend(infra.AddUnitOfWork).
      Extend(seatax.AddSeata).
      Extend(logx.AddLog(&logx.Config{Level: "info", Target: "both", Format: "text"})).
      Build()
  ```
- 事务内必须使用回调传入的 ctx（`Store.Context(ctx)` 自动命中当前事务）；嵌套 `UnitOfWork.Do` 由 gorm SAVEPOINT 复用外层事务。
- seatax 不包装初始化：先调用 seata-go/v2 的 `client.InitPath` 再打开数据源（代理驱动在初始化后才注册）。

## 验证

```sh
go test ./...
go vet ./...
```

## SKILL 使用指南

按任务检索源码（完整用法见各包 README）：

| 任务 | 包 | 关键符号 |
| --- | --- | --- |
| 业务错误 | errorx | `Define`、`CodeOf`、`ErrorOf`、`Wrap` |
| Gin 响应 | ginx | `Ok`、`Fail`、`FailParam`、`AutoErrorInterceptor` |
| gRPC 转换 | grpcx | `UnaryServerValidationInterceptor`、`ToGRPCError`、`FromGRPCError` |
| go-micro 转换 | microx | `ValidationHandlerWrapper`、`ErrorHandlerWrapper`、`ToMicroError`、`FromMicroError` |
| 数据访问/事务 | infra | `AddStore`、`AddUnitOfWork`、`NewStore`、`NewUnitOfWork` |
| 日志 | logx | `Config`、`NewInstance`、`AddLog` |
| 分布式事务 | seatax | `WithGlobalTx`、`GetSqlDb`、`GetGormDb`、`NewSeata`、`AddGormDb`、`AddSeata`、`GinTransactionMiddleware` |


