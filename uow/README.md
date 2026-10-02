# uow

本地（单库）事务的编排边界：把一次业务调用内的多次读写纳入同一个数据库事务。

## 核心入口

| 符号 | 说明 |
| --- | --- |
| `NewUnitOfWork(db)` | 构造本地事务边界 |
| `AddUnitOfWork` | 无参 `kernel.Extension`，`TryProvide` 幂等注册 |

## Do：本地事务编排

`Do` 在单个数据库事务内执行 `fn`，`fn` 返回 `nil` 提交、返回 `error` 回滚。事务提交成功后，依次执行 `afterCommit` 钩子——这是“事务外副作用”（发消息、清缓存）的正确位置，因为 Redis / MQ 不参与关系库事务：

```go
var uow uow.UnitOfWork = uow.NewUnitOfWork(db)
gdb := store.NewGormDB(db) // 业务侧持有数据访问入口

err := uow.Do(ctx, func(txCtx context.Context) error {
    return gdb.WithContext(txCtx).Create(&Order{...}).Error
}, func(ctx context.Context) error {
    // 事务已提交，安全发消息 / 清缓存
    return publisher.Publish(ctx, "order.created", payload)
})
```

嵌套 `Do` 由 gorm SAVEPOINT 复用外层事务。取库入口见 [store](../store)。
