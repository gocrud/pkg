# seatax

`seatax` 封装 [seata.apache.org/seata-go/v2](https://github.com/apache/incubator-seata-go)，同时提供 TM（全局事务）与 RM（AT / XA / TCC）能力，并为 gRPC、HTTP（gin）、go-micro 三种协议提供 XID 传播组件。所有 Seata 侧错误统一转换为 errorx 业务错误，业务错误原样透传，可直接复用 `ginx` / `grpcx` / `microx` 的统一错误处理链路。

## 错误码

| 错误码 | 含义 |
| --- | --- |
| `SEATA_BEGIN` | 全局事务开启失败 |
| `SEATA_COMMIT` | 全局事务提交失败 |
| `SEATA_ROLLBACK` | 全局事务回滚失败 |
| `SEATA_REGISTER` | 分支资源（TCC）注册失败 |
| `SEATA_XID_MISSING` | 严格模式下缺少 XID |
| `SEATA_CONFIG` | Seata 客户端初始化 / 配置错误 |
| `SEATA_INTERNAL` | 未识别的 Seata 内部错误 |

## 初始化

Seata 客户端通过 `seatago.yml` 配置（支持 yaml / yml / json / toml）。AT 模式依赖 `undo_log` 表，可用 `InitUndoLogMySQL` / `InitUndoLogPostgres` 初始化（见「场景五」）。Seata Server 地址与事务分组配置示例：

```yaml
# seatago.yml
server:
  service_group:
    default_tx_group: default
seata:
  service:
    vgroup-mapping:
      default_tx_group: default
    grouplist:
      default: 127.0.0.1:8091
  registry:
    type: file
  config:
    type: file
```

```go
if err := seatax.Init("./seatago.yml"); err != nil {
    return err
}
// 或使用内嵌配置
if err := seatax.InitFromConf(seatagoConf); err != nil {
    return err
}
```

`Init` 幂等，重复调用无副作用；配置缺失或非法时返回 `SEATA_CONFIG` 错误。注意：Seata 代理数据库驱动在 `Init` 成功后才注册，数据源必须初始化完成后再打开。

## 场景一：TM 全局事务

`WithGlobalTx` 按回调执行结果自动提交或回滚；业务错误码原样透传，回滚失败时保留业务错误码并附加 seata 二阶段失败作为 cause：

```go
package example

import (
    "context"
    "time"

    "github.com/gocrud/pkg/errorx"
    "github.com/gocrud/pkg/seatax"
)

func CreateOrder(ctx context.Context) error {
    return seatax.WithGlobalTx(ctx, "create-order", func(txCtx context.Context) error {
        // 扣库存、创建订单……任意一步返回错误即触发全局回滚
        if err := deductStock(txCtx); err != nil {
            return errorx.Define(errorx.ErrInternal, "扣减库存失败").Wrap(err)
        }
        return createOrderRecord(txCtx)
    }, seatax.WithTimeout(30*time.Second))
}
```

全局事务内通过 `seatax.GetXID(ctx)` 读取 XID；传播行为可用 `WithPropagation(seatax.PropagationNotSupported)` 等调整，锁重试可用 `WithLockRetry(interval, times)` 配置。

## 场景二：HTTP（gin）服务端

默认严格模式：请求未携带 XID 时返回 `SEATA_XID_MISSING` 业务错误（`c.Error` + `c.Abort`），由 `ginx.AutoErrorInterceptor` 统一渲染为 HTTP 200 的 `Result`；确需放行无 XID 请求时用 `WithAllowMissingXID()` 切换为宽松模式：

```go
package example

import (
    "github.com/gin-gonic/gin"
    "github.com/rs/zerolog"
    "github.com/gocrud/pkg/ginx"
    "github.com/gocrud/pkg/seatax"
)

func NewRouter(logger zerolog.Logger) *gin.Engine {
    r := gin.New()
    r.Use(seatax.GinTransactionMiddleware(), ginx.AutoErrorInterceptor(logger))
    r.POST("/order", func(c *gin.Context) {
        // handler 内 seatax.GetXID(c.Request.Context()) 即上游 XID
        ginx.Ok(c, nil)
    })
    return r
}
```

XID 头名称为 `TX_XID`（兼容小写 `tx_xid`）。上游通过 `seatax.InjectXIDHeader(req, xid)` 注入。gin >= 1.8.1 时若需通过 `c.Value()` 读取 seata 上下文，须将引擎的 `ContextWithFallback` 置为 true。

## 场景三：gRPC

服务端拦截器恢复 XID，客户端拦截器自动注入 XID，与 `grpcx` 校验拦截器组合使用：

```go
// 服务端
grpc.NewServer(grpc.ChainUnaryInterceptor(
    seatax.ServerTransactionInterceptor(),
    grpcx.UnaryServerValidationInterceptor(),
))

// 客户端
grpc.Dial(target,
    grpc.WithUnaryInterceptor(seatax.ClientTransactionInterceptor()),
    grpc.WithStreamInterceptor(seatax.ClientTransactionStreamInterceptor()))
```

## 场景四：go-micro

服务端 wrapper 恢复 XID，客户端 wrapper 注入 XID，与 `microx` 的校验、错误 wrapper 组合使用：

```go
micro.NewService(
    micro.WrapHandler(seatax.MicroServerTransactionWrapper(), microx.ValidationHandlerWrapper(), microx.ErrorHandlerWrapper()),
    micro.WrapClient(seatax.MicroClientTransactionWrapper()),
)
```

## 场景五：RM 数据源（AT / XA）

```go
db, err := seatax.OpenDataSource(seatax.ModeAT, seatax.DBTypeMySQL, "user:pass@tcp(127.0.0.1:3306)/db?parseTime=true")
db, err = seatax.OpenDataSource(seatax.ModeAT, seatax.DBTypePostgres, "host=127.0.0.1 user=postgres dbname=db sslmode=disable")
db, err = seatax.OpenDataSource(seatax.ModeXA, seatax.DBTypeMySQL, "user:pass@tcp(127.0.0.1:3306)/db?parseTime=true")
db, err = seatax.OpenDataSource(seatax.ModeXA, seatax.DBTypePostgres, "host=127.0.0.1 user=postgres dbname=db sslmode=disable")
```

返回标准 `*sql.DB`（代理驱动），可直接执行 SQL 或交给 `seatax.WrapGorm` 包装成 gorm（见「场景六」）。AT 模式依赖 `undo_log` 表；XA 模式要求数据库支持 XA 协议。驱动未注册（未调用 `Init`）时返回 `SEATA_CONFIG` 错误。

AT 模式可用 `seatax` 内置方法初始化 `undo_log` 表（0.3.0+ 含唯一索引 `ux_undo_log`），在业务库上执行一次即可（传入普通 `*gorm.DB`，重复调用幂等）：

```go
gdb, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
if err != nil {
    return err
}
if err := seatax.InitUndoLogMySQL(gdb); err != nil { // MySQL
    return err
}

pdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
if err != nil {
    return err
}
if err := seatax.InitUndoLogPostgres(pdb); err != nil { // PostgreSQL（字符串列使用 text）
    return err
}
```

## 场景六：RM gorm 与 IoC

与官方示例一致，先用 `OpenDataSource` 拿到 seata 代理 `*sql.DB`，再交给 `WrapGorm` 包装：

```go
sqlDB, err := seatax.OpenDataSource(seatax.ModeXA, seatax.DBTypeMySQL, dsn) // 官方 util.GetXAMySqlDb():返回 *sql.DB
gdb, err := seatax.WrapGorm(seatax.DBTypeMySQL, sqlDB)                       // 官方 gorm.Open(mysql.New(mysql.Config{Conn: sqlDB}), &gorm.Config{})

// 一行式便捷方法(等价于上面的两步)
gdb, err = seatax.OpenGorm(seatax.ModeAT, seatax.DBTypeMySQL, dsn)
gdb, err = seatax.OpenGorm(seatax.ModeXA, seatax.DBTypePostgres, dsn)
```

`WrapGorm` 的 `dbType` 与 `OpenDataSource` / `OpenGorm` 的 `mode` / `dbType` 仅支持 `DBTypeMySQL` / `DBTypePostgres` 常量（严格匹配，不兼容大小写变体或别名）；`WrapGorm` 的 `conn` 为空时返回 `ERR_PARAM`。

`AddDatabase` 与 `infra.AddDatabase` 同风格的 IoC 注册，把 seata 数据源注册为 `*gorm.DB` 单例，默认 AT 模式 + MySQL，可用 `WithMode` / `WithDBType` 调整：

```go
package example

import (
    "github.com/gocrud/ioc"
    "github.com/gocrud/pkg/seatax"
)

func Register(sc *ioc.ServiceCollection, dsn string) *ioc.ServiceCollection {
    // 选项式:默认 AT + MySQL,可用 WithMode / WithDBType 调整
    sc = seatax.AddDatabase(dsn, seatax.WithMode(seatax.ModeXA), seatax.WithDBType(seatax.DBTypePostgres))(sc)

    // 默认 AT + MySQL 无需选项
    sc = seatax.AddDatabase(dsn)(sc)
    return sc
}
```

## 场景七：TCC

```go
package example

import (
    "context"

    "github.com/gocrud/pkg/seatax"
)

type OrderTCC struct{}

func (*OrderTCC) GetActionName() string { return "orderTCC" }

func (*OrderTCC) Prepare(ctx context.Context, params interface{}) (bool, error) {
    // 一阶段资源预留
    return true, nil
}

func (*OrderTCC) Commit(ctx context.Context, bac *seatax.BusinessActionContext) (bool, error) {
    return true, nil
}

func (*OrderTCC) Rollback(ctx context.Context, bac *seatax.BusinessActionContext) (bool, error) {
    return true, nil
}

var orderProxy, _ = seatax.NewTCCProxy(&OrderTCC{}) // 注册失败返回 SEATA_REGISTER

func Reserve(ctx context.Context) error {
    return seatax.WithGlobalTx(ctx, "tcc-order", func(txCtx context.Context) error {
        _, err := orderProxy.Prepare(txCtx, reserveParams)
        return err
    })
}
```

## 错误处理兼容矩阵

| 场景 | seatax 行为 | 协议层渲染 |
| --- | --- | --- |
| 业务回调返回 errorx 业务错误 | 触发回滚后原样透传业务错误码 | `ginx` / `grpcx` / `microx` 按业务错误码正常渲染 |
| 业务回调 panic | 回滚后转换为 `ERR_SYS` 内部错误 | 协议层按内部错误渲染 |
| 开启 / 提交 / 回滚失败 | `SEATA_BEGIN` / `SEATA_COMMIT` / `SEATA_ROLLBACK` | 非 `ERR_*` 码按业务错误渲染（ginx 输出 HTTP 200 Result，grpcx/microx 走约定转换） |
| 回滚失败且业务失败 | 保留业务错误码，seata 失败作为 cause | 业务错误码渲染不受影响 |
| 严格模式缺 XID | `SEATA_XID_MISSING` | 与上同理，统一链路渲染 |
