package infra

import (
	"context"

	"github.com/gocrud/ioc"
	"gorm.io/gorm"
)

type txKey struct{}

type UnitOfWork struct {
	db *gorm.DB
}

func NewUnitOfWork(db *gorm.DB) *UnitOfWork {
	return &UnitOfWork{db: db}
}

func (unit *UnitOfWork) Execute(ctx context.Context, action func(txCtx context.Context) error) error {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok && tx != nil {
		return action(ctx)
	}
	return unit.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey{}, tx)
		return action(txCtx)
	})
}

func AddUnitOfWork() ioc.ServiceCollectionExtension {
	return func(sc *ioc.ServiceCollection) *ioc.ServiceCollection {
		sc.TryAddSingleton[*UnitOfWork](NewUnitOfWork)
		return sc
	}
}
