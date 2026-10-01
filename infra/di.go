package infra

import (
	"github.com/gocrud/kernel"
)

// AddStore 把 Store 注册为单例,依赖已注册的 *gorm.DB(由装配处注册)。
//
// 遵循 kernel 的扩展约定:只做注册、无返回值,内部使用 TryProvide 保证幂等,
// 既可单独 Extend,也可放进 []kernel.Extension 批量装配:
//
//	kernel.New().
//	    Extend(seatax.AddGormDb(seatax.ATMySQL, dsn)).
//	    Extend(infra.AddStore).
//	    Extend(infra.AddUnitOfWork)
func AddStore(b *kernel.AppBuilder) {
	b.TryProvide[Store](NewStore)
}

// AddUnitOfWork 把 UnitOfWork 注册为单例,依赖同上。
func AddUnitOfWork(b *kernel.AppBuilder) {
	b.TryProvide[UnitOfWork](NewUnitOfWork)
}
