package infra

import (
	"context"

	"gorm.io/gorm"
)

// UnitOfWork 是本地(单库)事务边界。与 Store 配合,把一次业务调用内的
// 多次读写纳入同一个数据库事务:Do 内 fn 收到的 ctx 已注入事务连接,
// 之后所有 store.Context(ctx) 都会命中该事务。
type UnitOfWork interface {
	// Do 在单个数据库事务内执行 fn。fn 返回 nil 提交,返回 error 回滚。
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

// uowImpl 是 UnitOfWork 的默认实现,包装根 *gorm.DB 用于开启事务。
type uowImpl struct {
	db *gorm.DB
}

// Do 实现 UnitOfWork。gorm 的 Transaction 已内置 SAVEPOINT,支持嵌套调用。
func (u *uowImpl) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, TxKey{}, tx))
	})
}

// NewUnitOfWork 以根连接 db 构造一个 UnitOfWork。
func NewUnitOfWork(db *gorm.DB) UnitOfWork {
	if db == nil {
		panic("infra: NewUnitOfWork 的 db 不能为空")
	}
	return &uowImpl{db: db}
}
