// Package uow 提供本地(单库)事务的编排边界 UnitOfWork:把一次业务调用内的
// 多次读写纳入同一个数据库事务,提交成功后执行 afterCommit 钩子。
//
// 导入路径为 github.com/gocrud/pkg/uow,包名与目录一致:
//
//	import "github.com/gocrud/pkg/uow"
//
// # 常用入口
//
//   - NewUnitOfWork(db) 返回本地事务边界;Do(ctx, fn, afterCommit...) 在单个
//     数据库事务内执行 fn,提交成功后依次执行 afterCommit 钩子(发消息/清缓存)。
//   - AddUnitOfWork 为无参扩展。
package uow
