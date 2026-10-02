// Package gormctx 定义事务 *gorm.DB 在 context 中的共享载体 TxKey。
//
// TxKey 是 store(取库)、uow(本地事务)与 seatax(分布式事务)三方之间的共享
// 契约:事务开启方通过 context.WithValue(ctx, gormctx.TxKey{}, tx) 注入连接,
// 取库方通过 ctx.Value(gormctx.TxKey{}) 命中连接。把 TxKey 单独成包,使三者
// 互不依赖。
//
// 导入路径为 github.com/gocrud/pkg/gormctx,包名与目录一致:
//
//	import "github.com/gocrud/pkg/gormctx"
package gormctx
