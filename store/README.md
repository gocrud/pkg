# store

数据访问入口：`GormDB` 按 ctx 取关系库连接、`Redis` 直取缓存客户端，并配套 kernel 依赖注入扩展。

## 核心入口

| 符号 | 说明 |
| --- | --- |
| `NewGormDB(db)` | 构造 `*GormDB`；`WithContext(ctx)` 按 ctx 命中事务连接并返回 `*gorm.DB` |
| `NewRedis(rdb)` | 构造 `*Redis`；`Client()` 返回 `redis.UniversalClient` |
| `AddDB(driver, dsn)` | 返回 `kernel.Extension`，按驱动（`MySQL`/`Postgres`）打开并注册 `*gorm.DB` |
| `AddGorm` | 无参 `kernel.Extension`，注册 `*GormDB`（依赖 `*gorm.DB`） |
| `AddRedis(cfg)` | 返回 `kernel.Extension`，注册 `redis.UniversalClient` 与 `*Redis` |

## GormDB：按 ctx 取库

`GormDB` 只负责“按 ctx 取库”，本身不携带事务语义。事务连接由 `uow.UnitOfWork`（本地事务）与 `seatax.Seata`（分布式事务）通过 `gormctx.TxKey` 注入 ctx：

- 无事务时，`WithContext(ctx)` 返回根连接；
- 处于 `uow.UnitOfWork.Do` 或 `seatax.Seata.Do` 内时，返回当前事务连接，使 repo 的每次读写自动落入同一事务。

Redis 无 ACID 事务、不随 ctx 变化，注入 `*store.Redis` 后直接 `Client()` 取用即可。

```go
// 业务 repo 只依赖用到的入口
type OrderRepo struct {
    db *store.GormDB
}

func (r *OrderRepo) Get(ctx context.Context, id int64) (*Order, error) {
    var o Order
    // 事务内自动命中事务连接
    if err := r.db.WithContext(ctx).First(&o, id).Error; err != nil {
        return nil, err
    }
    return &o, nil
}
```

## DI 注册

`GormDB` 与 `Redis` 是两条独立链路：只用关系库的模块只注册 `*GormDB`，`Build()` 不会因为缺少 Redis 而失败。

```go
// 关系库 + 本地事务，无需 Redis
app, err := kernel.New().
    Extend(store.AddDB(store.MySQL, dsn)). // 注册 *gorm.DB
    Extend(store.AddGorm).                     // 注册 *store.GormDB
    Extend(uow.AddUnitOfWork).
    Build()
```

```go
// 需要缓存时再加一条；需要 seata 代理连接时把 AddDB 换成 seatax.AddGormDb
app, err := kernel.New().
    Extend(seatax.AddGormDb(seatax.ATMySQL, dsn)).
    Extend(store.AddGorm).
    Extend(store.AddRedis(store.RedisConfig{Addr: "127.0.0.1:6379"})). // 注册 *store.Redis
    Extend(uow.AddUnitOfWork).
    Extend(seatax.AddSeata).
    Build()
```

`AddGorm` 依赖已注册的 `*gorm.DB`（`AddDB` 或 `seatax.AddGormDb` 提供）；`AddRedis` 自带客户端，不依赖其它组件。
