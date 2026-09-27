# infra

基于 GORM 的数据访问基础设施：数据库/Store/UnitOfWork 注册、基础模型、事务与工作单元。

## 数据库和基础模型

`AddDatabase(dsn, driver...)` 返回 IoC 注册扩展，注册 `*gorm.DB` 的单例工厂，工厂通过 `gorm.Open` 建立连接：

| driver | 行为 |
| --- | --- |
| 不传 | 默认 MySQL |
| `mysql` | MySQL |
| `pgsql`、`postgres`、`postgresql` | PostgreSQL |
| 空字符串、未知值或传入多个 driver | 工厂返回错误 |

driver 会去除首尾空格并转小写。DSN 使用对应 GORM 驱动格式，由应用从配置或环境变量读取。

`BaseModel` 提供：

| 字段 | 类型 | 语义 |
| --- | --- | --- |
| `ID` | `int64` | 自增主键 |
| `CreatedAt` | `int64` | 秒级创建时间，自动写入 |
| `UpdatedAt` | `int64` | 秒级更新时间，自动更新 |
| `DeletedAt` | `soft_delete.DeletedAt` | 软删除时间，0 表示未删除 |

可以嵌入业务持久化模型；表迁移、索引、连接池与连接关闭由应用负责。

## Store 与 UnitOfWork

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
        return errorx.Define(errorx.ErrParam, "数量必须大于零")
    }
    store := infra.NewStore(db)
    unit := infra.NewUnitOfWork(db)
    return unit.Execute(ctx, func(txCtx context.Context) error {
        var product Product
        err := store.ForUpdate(txCtx).First(&product, productID).Error
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return errorx.Define("PRODUCT_NOT_FOUND", "商品不存在").Wrap(err)
        }
        if err != nil {
            return errorx.Define(errorx.ErrInternal, "查询商品失败").Wrap(err)
        }
        if product.Stock < quantity {
            return errorx.Define("INSUFFICIENT_STOCK", "库存不足")
        }
        err = store.WithContext(txCtx).Model(&product).
            Update("stock", product.Stock-quantity).Error
        return errorx.Define(errorx.ErrInternal, "更新库存失败").Wrap(err)
    })
}
```

前提：传入已连接的 `*gorm.DB`，并已完成 `Product` 表迁移；示例不执行迁移。行锁效果取决于数据库及存储引擎的事务支持。

- `NewStore(db)` 和 `NewUnitOfWork(db)` 支持不使用 IoC 的直接构造。
- `Store.WithContext(ctx)` 返回携带当前 ctx 的新 session：优先使用 ctx 中的事务，没有事务则使用基础连接，不继承已有查询条件。
- `Store.ForUpdate(ctx)` 在有事务上下文时添加 `FOR UPDATE` 子句；无事务时返回普通查询，不添加锁子句，也不因缺少事务而报错。
- `UnitOfWork.Execute(ctx, action)` 在无事务时启动 GORM 事务：回调返回 nil 则提交，返回错误则回滚；panic 交由 GORM 回滚后继续传播。
- 嵌套 `Execute` 直接复用 ctx 中的已有事务，不创建新事务或 savepoint，只有最外层负责提交或回滚。内层返回错误必须继续向外返回，否则外层仍可能提交。
- 回调中的所有仓储操作必须传递 **`txCtx`**。传入原始 ctx 会脱离事务，绕过 Store 使用基础 db 也不会自动加入事务。
- 不进行跨库校验：即使 Store 或 UnitOfWork 配置了其他数据库，传入事务 ctx 后仍使用 ctx 中的事务连接，不会切换数据库。事务 ctx 不能在回调结束后继续使用。

## IoC 注册

| 注册扩展 | 服务类型 | 构造依赖 |
| --- | --- | --- |
| `infra.AddDatabase(dsn, driver...)` | `*gorm.DB` | DSN、驱动、数据库可用性 |
| `infra.AddStore()` | `*infra.Store` | `*gorm.DB` |
| `infra.AddUnitOfWork()` | `*infra.UnitOfWork` | `*gorm.DB` |

`Add...` 均返回 `ioc.ServiceCollectionExtension` 并使用 `TryAddSingleton` 注册；Store 与 UnitOfWork 应使用同一个数据库连接池。
