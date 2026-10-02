// Package store 提供数据访问聚合入口 Store:按 ctx 取关系库连接(GormDB)、
// 直取缓存客户端(Redis),并配套 kernel 依赖注入扩展(AddGormDB、AddStore、AddRedis)。
//
// 导入路径为 github.com/gocrud/pkg/store,包名与目录一致:
//
//	import "github.com/gocrud/pkg/store"
//
// # 常用入口
//
//   - NewStore(db, rdb) 返回聚合入口 Store;GormDB(ctx) 按 ctx 命中事务连接
//     (由 uow 或 seatax 注入),Redis() 返回缓存客户端(未配置时为 nil)。
//   - AddGormDB(driver, dsn) 返回 kernel.Extension,按驱动(MySQL/Postgres)打开
//     *gorm.DB 供 AddStore 注入;AddStore 为无参扩展;AddRedis(cfg) 返回
//     kernel.Extension,注册 redis.UniversalClient 供 AddStore 注入。
package store
