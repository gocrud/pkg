package uow

import "github.com/gocrud/kernel"

// AddUnitOfWork 把 UnitOfWork 注册为单例,依赖已注册的 *gorm.DB。
//
// 遵循 kernel 的扩展约定:只做注册、无返回值,内部使用 TryProvide 保证幂等,
// 既可单独 Extend,也可放进 []kernel.Extension 批量装配。
func AddUnitOfWork(b *kernel.AppBuilder) {
	b.TryProvide[UnitOfWork](NewUnitOfWork)
}
