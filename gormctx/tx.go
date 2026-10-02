package gormctx

// TxKey 是事务 *gorm.DB 在 context 中的载体。store.Store.GormDB 据此判断
// 当前是否处于事务中;uow.UnitOfWork(本地事务)与 seatax.Seata(分布式事务)
// 开启/注入事务时,把事务连接塞进派生 ctx。
//
// TxKey 是取库方(store)与事务开启方(uow / seatax)之间的共享契约,单独成包
// 使三方互不依赖。
type TxKey struct{}
