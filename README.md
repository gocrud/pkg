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
| `/infra` | `infra` | GORM 数据访问、Store、工作单元 | [infra/README.md](infra/README.md) |
| `/logx` | `logx` | zerolog 初始化与输出 | [logx/README.md](logx/README.md) |
| `/seatax` | `seatax` | Seata 分布式事务与 XID 传播 | [seatax/README.md](seatax/README.md) |

## 通用约定

- 错误统一走 `errorx`：`Define(code, msg)` 声明错误码，`CodeOf` / `ErrorOf` 提取业务码与消息；协议层通用码 `SUCCESS` / `ERR_PARAM` / `ERR_UNAUTH` / `ERR_SYS`。
- 协议出口（ginx / grpcx / microx）通过 `errorx.ErrorOf` 沿 `Unwrap` 链识别业务错误并渲染，内部错误不向客户端暴露原因。
- 依赖注入：`infra.Add*`、`logx.AddLog`、`seatax.AddDatabase` 均返回 `ioc.ServiceCollectionExtension`。
- 事务内必须使用回调传入的 `txCtx`；嵌套 `Execute` 复用外层事务。
- seatax 必须先 `Init` 再打开数据源（代理驱动在初始化后才注册）。

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
| 数据访问/事务 | infra | `AddDatabase`、`NewStore`、`NewUnitOfWork`、`Execute` |
| 日志 | logx | `Config`、`NewInstance`、`AddLog` |
| 分布式事务 | seatax | `Init`、`WithGlobalTx`、`OpenDataSource`、`WrapGorm`、`NewTCCProxy` |


