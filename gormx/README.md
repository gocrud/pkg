# gormx

基于 [gorm.io/gorm](https://gorm.io) 的模型基类。

## BaseModel

`BaseModel` 提供自增主键、秒级时间戳与软删除，业务模型内嵌即可：

```go
type Order struct {
    gormx.BaseModel
    // 业务字段...
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `ID` | `int64` | 自增主键 |
| `CreatedAt` | `int64` | 创建时间（秒级时间戳，`autoCreateTime`） |
| `UpdatedAt` | `int64` | 更新时间（秒级时间戳，`autoUpdateTime`） |
| `DeletedAt` | `soft_delete.DeletedAt` | 删除时间（`0` 未删除，`softDelete`） |

数据访问入口（按 ctx 取库）与本地事务分别在 [store](../store) 与 [uow](../uow)。
