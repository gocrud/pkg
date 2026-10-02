# store

数据访问聚合入口：按 ctx 取关系库连接、直取缓存客户端，并配套 kernel 依赖注入扩展。

## 核心入口

| 符号 | 说明 |
| --- | --- |
| `NewStore(db, rdb)` | 构造聚合入口 `Store`；`GormDB(ctx)` 按 ctx 命中事务连接，`Redis()` 返回缓存客户端 |
| `AddGormDB(driver, dsn)` | 返回 `kernel.Extension`，按驱动（`MySQL`/`Postgres`）打开并注册 `*gorm.DB` |
| `AddStore` | 无参 `kernel.Extension`，`TryProvide` 幂等注册 |
| `AddRedis(cfg)` | 返回 `kernel.Extension`，注册 `redis.UniversalClient` 供 `AddStore` 注入 |

## Store：数据访问入口

`Store` 只负责“按 ctx 取库”，本身不携带事务语义。事务连接由 `uow.UnitOfWork`（本地事务）与 `seatax.Seata`（分布式事务）通过 `gormctx.TxKey` 注入 ctx：

- 无事务时，`GormDB(ctx)` 返回根连接；
- 处于 `uow.UnitOfWork.Do` 或 `seatax.Seata.Do` 内时，返回当前事务连接，使 repo 的每次读写自动落入同一事务。

Redis 等其它资源无 ACID 事务，通过 `Redis()` 直取客户端即可；未配置时返回 `nil`。

```go
store := store.NewStore(db, rdb) // rdb 可为 nil

func (r *OrderRepo) Get(ctx context.Context, id int64) (*Order, error) {
    var o Order
    // 事务内自动命中事务连接
    if err := store.GormDB(ctx).First(&o, id).Error; err != nil {
        return nil, err
    }
    return &o, nil
}
```

## DI 注册

```go
app, err := kernel.New().
    Extend(seatax.AddGormDb(seatax.ATMySQL, dsn)).          // 注册 *gorm.DB
    Extend(store.AddRedis(store.RedisConfig{Addr: "127.0.0.1:6379"})).
    Extend(store.AddStore).                                 // 依赖 *gorm.DB + redis.UniversalClient
    Extend(uow.AddUnitOfWork).
    Extend(seatax.AddSeata).
    Build()
```

`AddStore` 依赖已注册的 `*gorm.DB` 与 `redis.UniversalClient`，两者缺一 `Build()` 即报错。非 Seata 场景用 `store.AddGormDB(store.MySQL, dsn)` 打开并注册普通连接。
