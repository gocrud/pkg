# seatax

`seatax` 封装 [seata.apache.org/seata-go/v2](https://github.com/apache/incubator-seata-go)，提供 TM（全局事务）与 RM（AT / XA 数据源）能力，并为 gin、gRPC、go-micro 三种协议提供 XID 传播组件。seatax 只做薄封装、不转换错误：业务回调返回的错误原样透传，可直接复用 `ginx` / `grpcx` / `microx` 的统一错误处理链路。

## 初始化

seatax 不包装客户端初始化，使用前需调用 seata-go/v2 的客户端初始化：

```go
import "seata.apache.org/seata-go/v2/pkg/client"

client.InitPath("./conf/client.yml")
// 或 client.Init()，使用默认配置
```

seata-go v2 的代理数据库驱动（AT / XA）在客户端初始化成功后才注册，因此数据源必须在初始化之后再打开。

## 场景一：TM 全局事务

`WithGlobalTx` 开启全局事务，按回调执行结果自动提交或回滚；`GtxConfig` / `CallbackWithCtx` 是 `tm` 包类型的别名：

```go
package example

import (
    "context"
    "time"

    "github.com/gocrud/pkg/errorx"
    "github.com/gocrud/pkg/seatax"
)

func CreateOrder(ctx context.Context) error {
    return seatax.WithGlobalTx(ctx, &seatax.GtxConfig{
        Name:    "create-order",
        Timeout: 30 * time.Second,
    }, func(txCtx context.Context) error {
        // 扣库存、创建订单……任意一步返回错误即触发全局回滚
        if err := deductStock(txCtx); err != nil {
            return errorx.Define("STOCK_DEDUCT_FAILED", "扣减库存失败").Wrap(err)
        }
        return createOrderRecord(txCtx)
    })
}
```

`GtxConfig` 字段：

| 字段 | 说明 |
| --- | --- |
| `Name` | 全局事务名，**必填**（空值直接返回错误） |
| `Timeout` | 全局事务超时 |
| `Propagation` | 传播行为，类型为 `tm.Propagation`（常量在 seata-go/v2 的 `tm` 包） |
| `LockRetryInternal` | 锁冲突重试间隔 |
| `LockRetryTimes` | 锁冲突重试次数 |

`gc` 为 `nil` 时同样返回错误。回调内通过 seata-go/v2 的 `tm.GetXID(ctx)` 读取当前 XID。

## 场景二：RM 数据源（AT / XA）

`GetSqlDb` 返回 seata 代理 `*sql.DB`，`GetGormDb` 一步得到 `*gorm.DB`（等价于 `GetSqlDb` + `gorm.Open`）：

```go
import (
    "github.com/gocrud/pkg/seatax"
    "gorm.io/gorm"
)

// 返回代理 *sql.DB,可直接执行 SQL 或交给 gorm.Open 包装
db, err := seatax.GetSqlDb(seatax.ATMySQL, "user:pass@tcp(127.0.0.1:3306)/db?parseTime=true")
db, err = seatax.GetSqlDb(seatax.ATPostgres, "host=127.0.0.1 user=postgres dbname=db sslmode=disable")
db, err = seatax.GetSqlDb(seatax.XAMySQL, "user:pass@tcp(127.0.0.1:3306)/db?parseTime=true")
db, err = seatax.GetSqlDb(seatax.XAPostgres, "host=127.0.0.1 user=postgres dbname=db sslmode=disable")

// 直接返回 *gorm.DB
var gdb *gorm.DB
gdb, err = seatax.GetGormDb(seatax.ATMySQL, "user:pass@tcp(127.0.0.1:3306)/db?parseTime=true")
```

`DriverName` 常量：

| 常量 | 模式 | 数据库 |
| --- | --- | --- |
| `ATMySQL` | AT | MySQL |
| `ATPostgres` | AT | PostgreSQL |
| `XAMySQL` | XA | MySQL |
| `XAPostgres` | XA | PostgreSQL |

AT 模式依赖业务库的 `undo_log` 表；XA 模式要求数据库支持 XA 协议。未初始化的驱动名或非法 driverName 会返回 `unsupported driver` 错误。

### undo_log 表（AT 模式）

AT 模式要求业务库中存在 `undo_log` 表，可直接执行随包提供的脚本，或用 `MigrateUndoLog` 交给 GORM 按方言建表（表不存在时创建，已存在时不做任何 DDL）：

| 数据库 | 建表脚本 | GORM 实体 |
| --- | --- | --- |
| MySQL | `undo_log.sql` | `seatax.UndoLog` |
| PostgreSQL | `undo_log_pg.sql` | `seatax.UndoLog` |

```go
gdb, err := seatax.GetGormDb(seatax.ATPostgres, dsn)
if err != nil {
    return err
}
if err := seatax.MigrateUndoLog(gdb); err != nil {
    return err
}
```

`UndoLog` 实体与两份脚本描述的表结构一致：字符串列（`xid` / `context` / `ext`）统一声明为 `text`；`rollback_info` 为 `[]byte`，由 GORM 按方言渲染为 `bytea`（PostgreSQL）/ `longblob`（MySQL）；`log_status` 为 `int32`，渲染为 `integer`（PostgreSQL）/ `int`（MySQL）；`id` 为 `bigserial`（PostgreSQL）/ `bigint auto_increment`（MySQL）；`xid + branch_id` 组成唯一索引 `ux_undo_log`（与脚本中的约束同名）；`log_created` 上的 `ix_log_created` 便于按时间清理历史记录。

`MigrateUndoLog` 只在表不存在时建表，不对既有表做 ALTER，因此不会删数据，也不会把脚本建好的列改写成等价但不同名的类型（如 `datetime` → `datetime(3)`）。如确需增量对齐实体与既有表，可自行调用 `db.AutoMigrate(&seatax.UndoLog{})`。

## 场景三：Seata 事务边界

`NewSeata` + `Do` 把 seata 代理的 `*gorm.DB` 注入 `infra.TxKey`，使 `infra.Store.Context(ctx)` 命中该连接；配合 `WithGlobalTx` 或协议层的 XID 传播，让仓库层的读写落入全局事务：

```go
package example

import (
    "context"

    "github.com/gocrud/pkg/infra"
    "github.com/gocrud/pkg/seatax"
    "gorm.io/gorm"
)

type OrderService struct {
    seata *seatax.Seata
    store infra.Store
}

func NewOrderService(db *gorm.DB) *OrderService {
    return &OrderService{
        seata: seatax.NewSeata(db),
        store: infra.NewStore(db),
    }
}

func (s *OrderService) Create(ctx context.Context) error {
    return seatax.WithGlobalTx(ctx, &seatax.GtxConfig{Name: "create-order"},
        func(txCtx context.Context) error {
            // Do 将代理连接注入 TxKey,store.Context 命中该连接
            return s.seata.Do(txCtx, func(dbCtx context.Context) error {
                return s.store.Context(dbCtx).Create(&Order{}).Error
            })
        })
}
```

`Do` 本身只注入连接、不开启全局事务，全局事务边界由 `WithGlobalTx` 或上游传播的 XID 决定。

## 场景四：XID 传播 —— HTTP（gin）

```go
package example

import (
    "github.com/gin-gonic/gin"
    "github.com/gocrud/pkg/ginx"
    "github.com/gocrud/pkg/seatax"
    "github.com/rs/zerolog"
)

func NewRouter(logger zerolog.Logger) *gin.Engine {
    r := gin.New()
    r.Use(seatax.GinTransactionMiddleware(), ginx.AutoErrorInterceptor(logger))
    r.POST("/order", func(c *gin.Context) {
        // c.Request.Context() 已携带上游 XID
        ginx.Ok(c, nil)
    })
    return r
}
```

中间件从请求头读取 XID（`TX_XID`，兼容小写 `tx_xid`）并写入请求上下文；缺少 XID 时以 HTTP 400 中断。使用 gin >= 1.8.1 时，若需通过 `c.Value()` 读取 seata 上下文，应将引擎的 `ContextWithFallback` 置为 true。

## 场景五：XID 传播 —— gRPC

服务端拦截器恢复 XID，客户端拦截器自动注入 XID，与 `grpcx` 校验拦截器组合使用：

```go
// 服务端
grpc.NewServer(grpc.ChainUnaryInterceptor(
    seatax.GrpcServerTransactionInterceptor(),
    grpcx.UnaryServerValidationInterceptor(),
))

// 客户端
grpc.NewClient(target,
    grpc.WithUnaryInterceptor(seatax.GrpcClientTransactionInterceptor()),
    grpc.WithStreamInterceptor(seatax.GrpcClientStreamTransactionInterceptor()))
```

## 场景六：XID 传播 —— go-micro

服务端 wrapper 恢复 XID，客户端 wrapper 注入 XID，与 `microx` 的校验、错误 wrapper 组合使用：

```go
service := micro.NewService("helloworld",
    micro.Server(grpcServer.NewServer(
        server.WrapHandler(
            seatax.MicroTransactionHandlerWrapper(),
            microx.ValidationHandlerWrapper(),
            microx.ErrorHandlerWrapper(),
        ),
    )),
    micro.WrapClient(
        seatax.MicroTransactionCallWrapper(),
    ),
)
```

## XID 传播机制要点

| 协议 | 组件 | 传播方式 |
| --- | --- | --- |
| HTTP（gin） | `GinTransactionMiddleware` | 请求头 `TX_XID`（兼容 `tx_xid`） |
| gRPC | `GrpcServerTransactionInterceptor` / `GrpcClientTransactionInterceptor` | gRPC metadata `TX_XID` |
| go-micro | `MicroTransactionHandlerWrapper` / `MicroTransactionCallWrapper` | go-micro metadata；grpc transport 会将键统一小写为 `tx_xid`，服务端两种写法均兼容 |

客户端组件只在当前上下文处于全局事务（`tm.IsSeataContext`）时注入 XID，否则原样透传；服务端组件在缺失 XID 时仅放行、不阻断（gin 例外，缺 XID 返回 400）。各协议均同时识别 `TX_XID` 与 `tx_xid`，以兼容大小写不一致的中间层。

## DI 注册

| 注册扩展 | 服务类型 | 构造依赖 |
| --- | --- | --- |
| `seatax.AddGormDb(driver, dsn)` | `*gorm.DB`（seata 代理连接） | 无（参数在注册时给定） |
| `seatax.AddSeata` | `*seatax.Seata` | `*gorm.DB` |

`AddGormDb` 需要参数、返回 `kernel.Extension`；`AddSeata` 本身就是 `kernel.Extension`。两者内部都用 `TryProvide` 注册（该类型/服务已注册时跳过），装配时用 `kernel.New().Extend(...)`：

```go
app, err := kernel.New().
    Extend(seatax.AddGormDb(seatax.ATMySQL, dsn)). // 注册 *gorm.DB
    Extend(seatax.AddSeata).                       // 依赖 *gorm.DB
    Build()
```

`AddGormDb` 不负责 seata 客户端初始化，仍需先 `client.InitPath` / `client.Init`，否则代理驱动未注册、打开数据源会失败。
