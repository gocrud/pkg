# gocrud/pkg

Go 服务端公共组件库，提供业务错误、Gin 响应、gRPC 错误转换、GORM 数据访问与事务、结构化日志。模块路径：`github.com/gocrud/pkg`。

本文同时面向开发者和读取项目上下文的 SKILL / AI Agent。示例以当前源码为准；本模块不包含业务服务入口、配置加载器或数据库迁移命令。

## 目录

- [环境与安装](#环境与安装)
- [包与源码索引](#包与源码索引)
- [业务错误 errorx](#业务错误-errorx)
- [HTTP 响应 ginx](#http-响应-ginx)
- [gRPC 校验与错误 grpcx](#grpc-校验与错误-grpcx)
- [数据库与事务 infra](#数据库与事务-infra)
- [日志 logx](#日志-logx)
- [依赖注入接入](#依赖注入接入)
- [SKILL 使用指南](#skill-使用指南)
- [验证](#验证)

## 环境与安装

- Go **1.27.0 或更高版本**，以 [go.mod](go.mod) 为准。
- 数据库支持 MySQL 和 PostgreSQL；只有使用数据库相关组件时才需要数据库实例。
- 主要依赖包括 Gin、gRPC、GORM、zerolog、lumberjack，以及 `github.com/gocrud/ioc` 和 `github.com/gocrud/veri`。

在消费方的 Go 模块中安装：

```sh
go get github.com/gocrud/pkg
```

生产项目应固定经过验证的 tag 或 commit。本文中的 Go 代码块是独立示例，不要将多个 `package main` 示例拼接进同一文件。

## 包与源码索引

| 导入路径后缀 | 实际包名 | 用途 | 源码入口 |
| --- | --- | --- | --- |
| `/errorx` | `errorx` | 业务错误、错误码、内部错误包装 | [errorx/error.go](errorx/error.go)、[errorx/codes.go](errorx/codes.go) |
| `/ginx` | **`httpx`** | Gin 统一响应、错误处理中间件 | [ginx/response.go](ginx/response.go)、[ginx/middleware_error.go](ginx/middleware_error.go) |
| `/grpcx` | `grpcx` | Unary 请求校验、服务端与客户端错误转换 | [grpcx/interceptors.go](grpcx/interceptors.go)、[grpcx/translator.go](grpcx/translator.go) |
| `/infra` | `infra` | 数据库注册、模型、Store、工作单元 | [infra/database.go](infra/database.go)、[infra/model.go](infra/model.go)、[infra/store.go](infra/store.go)、[infra/uow.go](infra/uow.go) |
| `/logx` | `logx` | zerolog 初始化、多目标输出、文件轮转 | [logx/config.go](logx/config.go)、[logx/log.go](logx/log.go)、[logx/writer.go](logx/writer.go)、[logx/ioc.go](logx/ioc.go) |

特别注意：导入 `github.com/gocrud/pkg/ginx` 后，默认标识符是 `httpx`。本文显式使用 `httpx` 别名，避免根据目录名误写调用。

## 业务错误 errorx

### 错误码

| 常量 | 值 | 含义 |
| --- | --- | --- |
| `errorx.ErrOK` | `SUCCESS` | 成功 |
| `errorx.ErrInternal` | `ERR_SYS` | 内部错误 |
| `errorx.ErrParam` | `ERR_PARAM` | 参数错误 |

业务可以自定义字符串错误码，例如 `USER_NOT_FOUND`。

### 创建和包装

```go
package example

import (
    "errors"

    "github.com/gocrud/pkg/errorx"
    "gorm.io/gorm"
)

func TranslateLookupError(err error) error {
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return errorx.E("USER_NOT_FOUND", "用户不存在", err)
    }
    return errorx.Wrap(err, "查询用户失败")
}
```

- `E(code, userMsg, cause...)` 返回一个 `BizError` 值，仅保存第一个可选 cause；可通过 `errors.Unwrap` 或 `errors.Is` 访问错误链。
- `Wrap(nil, msg)` 返回 `nil`。
- `Wrap` 用 `errors.As` 查找 **`BizError` 值类型**；找到后原样返回传入错误，否则包装为 `ERR_SYS`。建议通过 `E` / `Wrap` 构造错误，避免自行使用 `*BizError` 导致识别行为不同。
- `Wrap` 的 `StackStr()` 是包装调用位置的 `文件名:行号`，不是完整堆栈；`E` 不记录调用位置。
- `BizErrorBehavior` 约定 `CodeStr()` 与 `MsgStr()`。HTTP 中间件还要求 `StackStr()`，两种协议的识别条件并不相同。
- 面向用户的消息不要包含 SQL、连接串、凭据或内部堆栈。内部 cause 留给日志处理。

## HTTP 响应 ginx

### 最小服务

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/gocrud/pkg/errorx"
    httpx "github.com/gocrud/pkg/ginx"
    "github.com/gocrud/pkg/logx"
)

func main() {
    logger := logx.NewInstance(&logx.Config{
        Level: "info", Target: "console", Format: "text",
    })
    router := gin.New()
    router.Use(gin.Recovery(), httpx.AutoErrorInterceptor(logger))

    router.GET("/health", func(ctx *gin.Context) {
        httpx.Ok(ctx, gin.H{"status": "ok"})
    })
    router.GET("/users/:id", func(ctx *gin.Context) {
        httpx.Fail(ctx, errorx.E("USER_NOT_FOUND", "用户不存在"))
        return
    })

    if err := router.Run(":8080"); err != nil {
        logger.Fatal().Err(err).Msg("http server stopped")
    }
}
```

### 响应接口

| API | 行为 |
| --- | --- |
| `Ok(ctx, data)` | HTTP 200，`code=SUCCESS`、`msg=success`，携带 data |
| `Msg(ctx, msg)` | HTTP 200，`code=SUCCESS`，自定义 msg |
| `FailParam(ctx)` | 登记 `ERR_PARAM`，消息为“请求参数格式错误”，并 Abort |
| `Fail(ctx, err)` | err 非 nil 时登记错误并 Abort；nil 时不做任何事 |
| `AutoErrorInterceptor(logger)` | 下游执行后，将最后一条登记错误转换为响应 |

`Fail` 和 `FailParam` 本身不写 JSON，必须搭配中间件。`Abort()` 不会退出当前 Go 函数，因此调用后通常需要 `return`。使用 `ShouldBindJSON` 等绑定方法时，应自行检查返回错误，再调用 `FailParam` 或 `Fail`。

统一响应结构为 `Result`：`code`、`msg`，以及带 `omitempty` 的 `data`、`errors`。参数校验响应示例：

```json
{
  "code": "ERR_PARAM",
  "msg": "参数校验未通过",
  "errors": [{"field": "email", "message": "邮箱格式不正确"}]
}
```

### 错误映射与边界

| 错误类型 | HTTP 状态 | 响应 |
| --- | --- | --- |
| 错误链包含 `*veri.ValidationErrors` | 200 | `ERR_PARAM`，保留字段与消息列表 |
| 顶层错误实现 `CodeStr/MsgStr/StackStr`，且 code 非 `ERR_SYS` | 200 | 业务 code 与用户消息 |
| 顶层错误实现上述接口，且 code 为 `ERR_SYS` | 500 | `ERR_SYS`，固定消息“系统繁忙，请稍后再试” |
| 其他错误 | 500 | `ERR_SYS`，固定消息“服务异常” |

- 业务错误分支使用顶层类型断言，不沿错误链查找。不要在传给 `Fail` 之前用 `fmt.Errorf("...: %w", bizErr)` 再包装业务错误，否则会按未知错误处理。
- 只处理 `ctx.Errors` 的最后一条错误；若响应已写入，则跳过处理。
- 中间件不主动执行参数校验，也不负责 panic 恢复。`gin.Recovery()` 的响应不保证符合 `Result` 结构。

## gRPC 校验与错误 grpcx

### 服务端接入

```go
package example

import (
    "context"

    "github.com/gocrud/pkg/grpcx"
    "google.golang.org/grpc"
)

func NewServer() *grpc.Server {
    return grpc.NewServer(grpc.ChainUnaryInterceptor(
        grpcx.UnaryServerValidationInterceptor("/grpc.health.v1.Health/Check"),
        func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
            response, err := handler(ctx, req)
            return response, grpcx.HandleServerError(ctx, err)
        },
    ))
}
```

上面的第二个拦截器是消费方示例代码，不是本库导出的 API。它统一转换 handler 返回的错误；也可以不添加它，改为在每个 RPC 方法中显式调用 `HandleServerError`。两种方式选一种，避免重复转换。

`UnaryServerValidationInterceptor(skipMethods...)`：

- 仅用于 Unary RPC，不支持流式 RPC 校验。
- 请求实现 `Validate() error` 时执行校验，否则直接放行。
- 跳过列表精确匹配 `info.FullMethod`，格式为 `/package.Service/Method`。
- 只转换校验阶段返回的错误，**不会转换 handler 返回的错误**。

### 协议映射

| 服务端错误 | gRPC code | trailer |
| --- | --- | --- |
| `nil` | 无错误 | 不设置 |
| 错误链包含 `*veri.ValidationErrors` | `InvalidArgument` | `ERR_PARAM`、固定原因、字段详情 |
| 错误链包含 `CodeStr/MsgStr`，且 code 非 `ERR_SYS` | `Aborted` | 业务 code 与用户消息 |
| 其他错误，包括 `ERR_SYS` | `Internal` | `ERR_SYS`、固定原因 |

Trailer 键为 `HeaderBizCode` (`x-biz-code`)、`HeaderBizReason` (`x-biz-reason`)、`HeaderBizDetail` (`x-biz-detail`)。字段详情采用 `field:message; field:message` 文本，不是 JSON。

注意：手工创建 `errorx.E(errorx.ErrParam, ...)` 属于普通业务错误，在 gRPC 中映射为 `Aborted`；只有 `*veri.ValidationErrors` 分支映射为 `InvalidArgument`。已有 gRPC status 错误若不满足业务错误条件，也会被转换为 `Internal`，不会原样透传。`HandleServerError` 不记录日志，应由服务端自行记录内部原因。

### 客户端还原

```go
package example

import (
    "context"

    "github.com/gocrud/pkg/grpcx"
    "google.golang.org/grpc"
    "google.golang.org/grpc/metadata"
)

func Invoke(ctx context.Context, conn grpc.ClientConnInterface, method string, request, response any) error {
    var trailer metadata.MD
    err := conn.Invoke(ctx, method, request, response, grpc.Trailer(&trailer))
    _, translated := grpcx.HandleClientError(err, trailer)
    return translated
}
```

生成的客户端方法同样可以追加 `grpc.Trailer(&trailer)` 调用选项。`HandleClientError` 返回 `(converted bool, err error)`：仅对 `Aborted`、`Internal`、`InvalidArgument` 且存在非空 `x-biz-code` 的错误返回 `true` 并构造 `errorx.E`；其他错误原样返回，nil 返回 `(false, nil)`。当前不会还原 `x-biz-detail`，也不会保留原 gRPC 错误作为 cause。

## 数据库与事务 infra

### 数据库和基础模型

`AddDatabase(dsn, driver...)` 返回 IoC 注册扩展，注册 `*gorm.DB` 的单例工厂，工厂通过 `gorm.Open` 建立连接：

| driver | 行为 |
| --- | --- |
| 不传 | 默认 MySQL |
| `mysql` | MySQL |
| `pgsql`、`postgres`、`postgresql` | PostgreSQL |
| 空字符串、未知值或传入多个 driver | 工厂返回错误 |

driver 会去除首尾空格并转小写。DSN 使用对应 GORM 驱动格式，从消费方的配置或环境变量读取；本库不读取环境变量、不配置连接池、不迁移表结构，也不提供统一资源关闭入口。

`BaseModel` 提供：

| 字段 | 类型 | 语义 |
| --- | --- | --- |
| `ID` | `int64` | 自增主键 |
| `CreatedAt` | `int64` | 秒级创建时间，自动写入 |
| `UpdatedAt` | `int64` | 秒级更新时间，自动更新 |
| `DeletedAt` | `soft_delete.DeletedAt` | 软删除时间，0 表示未删除 |

可以嵌入业务持久化模型。迁移、索引和连接池策略由应用负责；正常关闭服务时，由应用取得底层 `*sql.DB` 并管理其关闭。

### Store 与 UnitOfWork

```go
package example

import (
    "context"
    "errors"

    "github.com/gocrud/pkg/errorx"
    "github.com/gocrud/pkg/infra"
    "gorm.io/gorm"
)

type Product struct {
    infra.BaseModel
    Stock int64
}

func Reserve(ctx context.Context, db *gorm.DB, productID, quantity int64) error {
    if quantity <= 0 {
        return errorx.E(errorx.ErrParam, "数量必须大于零")
    }
    store := infra.NewStore(db)
    unit := infra.NewUnitOfWork(db)
    return unit.Execute(ctx, func(txCtx context.Context) error {
        var product Product
        err := store.ForUpdate(txCtx).First(&product, productID).Error
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return errorx.E("PRODUCT_NOT_FOUND", "商品不存在", err)
        }
        if err != nil {
            return errorx.Wrap(err, "查询商品失败")
        }
        if product.Stock < quantity {
            return errorx.E("INSUFFICIENT_STOCK", "库存不足")
        }
        err = store.WithContext(txCtx).Model(&product).
            Update("stock", product.Stock-quantity).Error
        return errorx.Wrap(err, "更新库存失败")
    })
}
```

前提：传入已连接的 `*gorm.DB`，并已完成 `Product` 表迁移；示例不执行迁移。行锁效果取决于数据库及存储引擎的事务支持。

- `NewStore(db)` 和 `NewUnitOfWork(db)` 支持不使用 IoC 的直接构造。
- `Store.WithContext(ctx)` 优先使用 ctx 中的事务，没有事务则使用基础连接。
- `Store.ForUpdate(ctx)` 要求事务上下文，并添加 `FOR UPDATE` 子句；无事务时返回的 `*gorm.DB.Error` 为 `ErrTransactionScope`。
- `UnitOfWork.Execute(ctx, action)` 在无事务时启动 GORM 事务：回调返回 nil 则提交，返回错误则回滚；panic 交由 GORM 回滚后继续传播。
- 同一连接池中的嵌套 `Execute` 复用已有事务，不创建新事务或 savepoint。内层返回错误必须继续向外返回，否则外层仍可能提交。
- 回调中的所有仓储操作必须传递 **`txCtx`**。传入原始 ctx 会脱离事务，绕过 Store 使用基础 db 也不会自动加入事务。
- ctx 内绑定的事务不能跨不同的底层 `*sql.DB` 使用，也不能在回调结束后继续使用。

### 底层事务 API

`TxOrDB(ctx, base, mode)` 返回携带当前 context 的新 GORM session：

| mode | 无事务 | 有同库事务 |
| --- | --- | --- |
| `TxOptional` | 基础连接 | 已有事务 |
| `TxRequired` | `ErrTransactionScope` | 已有事务 |
| `TxForbidden` | 基础连接 | `ErrTransactionScope` |

跨库事务绑定或非法 mode 同样通过返回值的 `.Error` 报错。调用方必须检查 `.Error`，可以用 `errors.Is(err, infra.ErrTransactionScope)` 判断范围错误。

`WithTx(ctx, base, tx)` 只负责把事务绑定到 context，不启动、提交或回滚事务。已有绑定或 tx 为 nil 时拒绝绑定；调用方必须保证 base 有效且 tx 确实来自 base，函数不会验证 tx 的真实来源。常规业务优先使用 `UnitOfWork.Execute`。

## 日志 logx

`NewInstance(cfg *Config)` 返回 **`zerolog.Logger` 值类型**。cfg 必须非 nil；配置加载由应用负责。

| Config 字段 | 配置键 | 行为 |
| --- | --- | --- |
| `Level` | `level` | zerolog 等级，例如 `debug`、`info`、`warn`、`error`；解析失败回退到 info |
| `Target` | `target` | `console`、`file`、`both`；其他值回退到标准输出 |
| `Format` | `format` | 仅在 console 分支中，`text` 启用可读格式；其他值为 JSON |
| `FilePath` | `file_path` | 文件路径，交给 lumberjack 处理 |
| `MaxBackups` | `max_backups` | 保留旧文件数量，交给 lumberjack 处理 |
| `MaxSize` | `max_size` | 单文件最大大小，单位 MB，交给 lumberjack 处理 |

字段同时提供 `mapstructure` 和 `json` 标签。文件输出始终为 JSON，并启用 `LocalTime` 与 `Compress`；例如 `Target=both, Format=text` 表示控制台可读文本、文件 JSON。Target / Format 按精确字符串匹配，不自动规范化。

配置示例，需由消费方解析到 `logx.Config`：

```json
{
  "level": "info",
  "target": "both",
  "format": "text",
  "file_path": "logs/app.log",
  "max_backups": 7,
  "max_size": 100
}
```

日志自动包含 timestamp 和 caller。`NewInstance` 会将全局 `zerolog.TimeFieldFormat` 设置为 `time.RFC3339Nano`，建议在启动阶段完成初始化。返回值不暴露统一的 Close 方法。

## 依赖注入接入

所有 `Add...` 函数均返回 `ioc.ServiceCollectionExtension`，并使用 `TryAddSingleton` 注册。本示例接收应用已有的 collection，直接应用扩展，不依赖猜测的容器构建 API：

```go
package example

import (
    "github.com/gocrud/ioc"
    "github.com/gocrud/pkg/infra"
    "github.com/gocrud/pkg/logx"
)

func Register(sc *ioc.ServiceCollection, dsn string, cfg *logx.Config) *ioc.ServiceCollection {
    extensions := []ioc.ServiceCollectionExtension{
        logx.AddLog(cfg),
        infra.AddDatabase(dsn, "postgres"),
        infra.AddStore(),
        infra.AddUnitOfWork(),
    }
    for _, extension := range extensions {
        sc = extension(sc)
    }
    return sc
}
```

| 注册扩展 | 服务类型 | 构造依赖 |
| --- | --- | --- |
| `logx.AddLog(cfg)` | `zerolog.Logger` | 非 nil 的 cfg |
| `infra.AddDatabase(dsn, driver...)` | `*gorm.DB` | DSN、驱动、数据库可用性 |
| `infra.AddStore()` | `*infra.Store` | `*gorm.DB` |
| `infra.AddUnitOfWork()` | `*infra.UnitOfWork` | `*gorm.DB` |

容器创建、构建、解析、错误处理及重复注册的具体规则，以项目锁定版本的 `github.com/gocrud/ioc` 文档为准。Store 与 UnitOfWork 应使用同一个数据库连接池。

## SKILL 使用指南

本节是供 SKILL 引用的知识入口，**README 本身不是可自动发现或执行的 SKILL**。在已有 SKILL 中引用本文件，并根据任务按需加载下表中的源码；不要把整个模块实现复制进技能指令。

### 按任务检索

| 任务关键词 | 优先阅读 | 关键符号 |
| --- | --- | --- |
| 业务错误、用户提示、cause | [errorx/error.go](errorx/error.go) | `E`、`Wrap`、`BizError` |
| Gin、HTTP、统一响应、参数错误 | [ginx/response.go](ginx/response.go)、[ginx/middleware_error.go](ginx/middleware_error.go) | `httpx.Ok`、`Fail`、`FailParam`、`AutoErrorInterceptor` |
| RPC、Validate、trailer、错误还原 | [grpcx/interceptors.go](grpcx/interceptors.go)、[grpcx/translator.go](grpcx/translator.go) | `UnaryServerValidationInterceptor`、`HandleServerError`、`HandleClientError` |
| GORM、MySQL、PostgreSQL、连接注册 | [infra/database.go](infra/database.go) | `AddDatabase` |
| 事务、行锁、仓储、工作单元 | [infra/uow.go](infra/uow.go)、[infra/store.go](infra/store.go) | `Execute`、`WithContext`、`ForUpdate`、`TxOrDB` |
| 持久化模型、时间戳、软删除 | [infra/model.go](infra/model.go) | `BaseModel` |
| 日志、轮转、控制台、文件 | [logx/config.go](logx/config.go)、[logx/writer.go](logx/writer.go)、[logx/log.go](logx/log.go) | `Config`、`NewInstance` |
| IoC、单例注册 | [logx/ioc.go](logx/ioc.go)、[infra/database.go](infra/database.go)、[infra/store.go](infra/store.go)、[infra/uow.go](infra/uow.go) | `AddLog`、`AddDatabase`、`AddStore`、`AddUnitOfWork` |

### 执行约束

1. 先检查目标项目的 Go 版本、锁定依赖和现有注册方式，再选择本模块的 API。
2. 以源码为准，不虚构 Redis 封装、配置加载器、通用 CRUD、流式 RPC 拦截器或自动迁移功能；依赖中出现某个库不代表已有对应功能。
3. HTTP 导入路径是 `/ginx`，包标识符是 `httpx`；错误响应依赖 `AutoErrorInterceptor`。
4. 区分 HTTP 顶层业务错误断言与 gRPC 的错误链识别；不要统一假设它们都支持任意包装。
5. gRPC 校验与 handler 错误转换是两步，客户端还原必须取得 trailer。
6. 事务内使用 txCtx；行锁必须在事务内；不要把嵌套 Execute 当成独立提交或局部回滚。
7. 使用非 nil 的日志配置，显式选择输出目标；不在示例或日志里写真实凭据。
8. 修改后执行与改动相关的检查，明确区分编译通过与真实数据库 / RPC 集成验证通过。

可在已有 SKILL 中使用以下任务描述作为引用模板，路径按消费方工作区调整：

```text
使用 github.com/gocrud/pkg 接入当前服务。
先阅读该模块 README 的“SKILL 使用指南”，按任务索引读取相关源码。
确认目标项目 Go 版本、依赖版本、现有 IoC 注册和协议约定。
只使用源码中存在的 API，遵守错误映射与事务上下文约束。
沿用目标项目的测试方式，报告实际执行的验证及未验证项。
```

## 验证

在模块根目录执行：

```sh
go test ./...
go vet ./...
```

当前模块未提供测试文件。上述命令不能替代集成验证；接入应用后应覆盖业务错误响应、gRPC trailer 转换、事务回滚、跨库拒绝、无事务行锁拒绝及日志文件轮转。本文数据库示例依赖应用提供连接和表结构，gRPC 示例依赖应用注册服务及建立客户端连接。