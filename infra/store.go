package infra

import (
	"context"

	"gorm.io/gorm"
)

// txKey 是事务 *gorm.DB 在 context 中的载体。UnitOfWork.Do 开启事务后,
// 把事务连接塞进派生 ctx;Store.Context 据此判断当前是否处于本地事务中。
type TxKey struct{}

// Store 是数据访问的底层入口,只负责"按 ctx 取库",本身不携带事务语义。
//
// 事务能力由 UnitOfWork(本地事务)与 seatax.Seata(分布式事务)通过 ctx 注入:
//   - 无事务时,Context(ctx) 返回根连接;
//   - 处于 UnitOfWork.Do 内时,返回当前事务连接,使 repo 的每次读写自动落入同一事务。
//
// Store 无状态、可复用,业务 repo 层统一依赖本接口而非 *gorm.DB。
type Store interface {
	// Context 返回绑定 ctx 的 *gorm.DB。若 ctx 内已由 UnitOfWork 注入事务
	// 连接,则返回该事务连接,否则返回根连接。
	Context(ctx context.Context) *gorm.DB
}

// storeImpl 是 Store 的默认实现,包装一个根 *gorm.DB。
type storeImpl struct {
	db *gorm.DB
}

// Context 实现 Store。
func (s *storeImpl) Context(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(TxKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return s.db.WithContext(ctx)
}

// NewStore 以根连接 db 构造一个 Store。
func NewStore(db *gorm.DB) Store {
	if db == nil {
		panic("infra: NewStore 的 db 不能为空")
	}
	return &storeImpl{db: db}
}
