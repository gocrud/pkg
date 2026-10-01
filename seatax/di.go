package seatax

import (
	"github.com/gocrud/kernel"
	"gorm.io/gorm"
)

// AddGormDb 返回一个 kernel 扩展,把 seata 代理驱动打开的 *gorm.DB 注册为单例。
// 数据源必须在 seata 客户端初始化成功之后再打开(代理驱动在初始化后才注册):
//
//	kernel.New().Extend(seatax.AddGormDb(seatax.ATMySQL, dsn))
func AddGormDb(driverName DriverName, dataSourceName string) kernel.Extension {
	return func(b *kernel.AppBuilder) {
		b.TryProvide[*gorm.DB](func() (*gorm.DB, error) {
			return GetGormDb(driverName, dataSourceName)
		})
	}
}

// AddSeata 把 *Seata 注册为单例,依赖已注册的 *gorm.DB(应为 seata 代理连接)。
func AddSeata(b *kernel.AppBuilder) {
	b.TryProvide[*Seata](NewSeata)
}
