// Package infra 提供基于 GORM 的数据访问基础设施。
//
// # 组成
//
//   - BaseModel:ID(int64 自增主键)、CreatedAt / UpdatedAt(int64 秒级)、
//     DeletedAt(soft_delete.DeletedAt,0 表示未删除)。
//   - Store:NewStore(db);WithContext(ctx) 优先使用 ctx 中的事务,否则用基础连接;
//     ForUpdate(ctx) 在有事务时添加 FOR UPDATE 行锁。
//   - UnitOfWork:NewUnitOfWork(db);Execute(ctx, action) 在无事务时开启事务,
//     返回 nil 提交、返回错误回滚;嵌套调用复用外层事务,仅最外层负责提交或回滚。
//   - IoC 扩展:AddDatabase(dsn, driver...)、AddStore()、AddUnitOfWork(),
//     以 TryAddSingleton 注册 *gorm.DB / *infra.Store / *infra.UnitOfWork。
//
// 回调内必须使用传入的 txCtx;表迁移、索引、连接池与关闭由应用负责。详见本包 README。
package infra
