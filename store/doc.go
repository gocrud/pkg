// Package store 提供数据访问入口:GormDB 按 ctx 取关系库连接(自动命中 uow /
// seatax 注入的事务),Redis 直取缓存客户端;两者各配一个 kernel 依赖注入扩展
// (AddGorm / AddRedis),互不依赖——只用关系库的模块不必注册 Redis。
//
// 导入路径为 github.com/gocrud/pkg/store,包名与目录一致:
//
//	import "github.com/gocrud/pkg/store"
//
// # 常用入口
//
//   - NewGormDB(db) 构造 GormDB;WithContext(ctx) 按 ctx 取库。
//   - NewRedis(rdb) 构造 Redis;Client() 取缓存客户端。
//   - AddDB(driver, dsn) 与 AddGorm 注册 *gorm.DB 与 *GormDB;
//     AddRedis(cfg) 注册 redis.UniversalClient 与 *Redis。
package store
