package uow

import (
	"context"

	"github.com/gocrud/pkg/gormctx"
	"gorm.io/gorm"
)

// UnitOfWork 是本地(单库)事务的编排边界,把一次业务调用内的多次读写纳入
// 同一个数据库事务。事务只覆盖关系库的 ACID 语义;Redis / MQ 等资源不参与,
// 应在提交成功后通过 afterCommit 钩子执行。
type UnitOfWork interface {
	// Do 在单个数据库事务内执行 fn:返回 nil 提交并依次执行 afterCommit 钩子,
	// 返回 error 回滚;钩子返回 error 时事务已提交,该错误直接返回。
	Do(ctx context.Context, fn func(ctx context.Context) error, afterCommit ...func(ctx context.Context) error) error
}

type uowImpl struct {
	db *gorm.DB
}

// Do 实现 UnitOfWork。嵌套调用由 gorm 的 SAVEPOINT 支持。
func (u *uowImpl) Do(ctx context.Context, fn func(ctx context.Context) error, afterCommit ...func(ctx context.Context) error) error {
	if err := u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, gormctx.TxKey{}, tx))
	}); err != nil {
		return err
	}
	for _, hook := range afterCommit {
		if hook == nil {
			continue
		}
		if err := hook(ctx); err != nil {
			return err
		}
	}
	return nil
}

// NewUnitOfWork 以根连接 db 构造 UnitOfWork。
func NewUnitOfWork(db *gorm.DB) UnitOfWork {
	if db == nil {
		panic("uow: NewUnitOfWork 的 db 不能为空")
	}
	return &uowImpl{db: db}
}
