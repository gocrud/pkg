# gormctx

事务 `*gorm.DB` 在 context 中的共享载体，只定义一个类型 `TxKey`。

`TxKey` 是取库方（`store`）与事务开启方（`uow` 本地事务、`seatax` 分布式事务）之间的共享契约：

- 事务开启方：`context.WithValue(ctx, gormctx.TxKey{}, tx)`
- 取库方：`ctx.Value(gormctx.TxKey{}).(*gorm.DB)`

单独成包是为了让三方互不依赖——`seatax` 只需 import `gormctx`，不必连带 `store`/`uow`。
