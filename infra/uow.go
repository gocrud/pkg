package infra

import (
	"context"
	"database/sql"
	"errors"

	"github.com/gocrud/ioc"
	"gorm.io/gorm"
)

type txKey struct{}

var ErrTransactionScope = errors.New("transaction scope mismatch or nested transaction binding")

type txScope struct {
	db *sql.DB
	tx *gorm.DB
}

func hasTx(ctx context.Context) bool {
	_, ok := ctx.Value(txKey{}).(*txScope)
	return ok
}

type TxMode uint8

const (
	TxOptional TxMode = iota
	TxRequired
	TxForbidden
)

func WithTx(ctx context.Context, base, tx *gorm.DB) (context.Context, error) {
	if hasTx(ctx) || tx == nil {
		return nil, ErrTransactionScope
	}
	pool, err := base.DB()
	if err != nil {
		return nil, err
	}
	return context.WithValue(ctx, txKey{}, &txScope{db: pool, tx: tx}), nil
}

type UnitOfWork struct {
	db *gorm.DB
}

func NewUnitOfWork(db *gorm.DB) *UnitOfWork {
	return &UnitOfWork{db: db}
}

func (unit *UnitOfWork) Execute(ctx context.Context, action func(txCtx context.Context) error) error {
	if hasTx(ctx) {
		if err := TxOrDB(ctx, unit.db, TxRequired).Error; err != nil {
			return err
		}
		return action(ctx)
	}
	return unit.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx, err := WithTx(ctx, unit.db, tx)
		if err != nil {
			return err
		}
		return action(txCtx)
	})
}

func TxOrDB(ctx context.Context, base *gorm.DB, mode TxMode) *gorm.DB {
	fail := func() *gorm.DB {
		session := base.Session(&gorm.Session{NewDB: true, Context: ctx})
		session.AddError(ErrTransactionScope)
		return session
	}
	if mode > TxForbidden {
		return fail()
	}
	scope, exists := ctx.Value(txKey{}).(*txScope)
	if exists {
		if mode == TxForbidden || scope == nil || scope.tx == nil {
			return fail()
		}
		pool, err := base.DB()
		if err != nil || pool != scope.db {
			return fail()
		}
		return scope.tx.Session(&gorm.Session{NewDB: true, Context: ctx})
	}
	if mode == TxRequired {
		return fail()
	}
	return base.Session(&gorm.Session{NewDB: true, Context: ctx})
}

func AddUnitOfWork() ioc.ServiceCollectionExtension {
	return func(sc *ioc.ServiceCollection) *ioc.ServiceCollection {
		sc.TryAddSingleton[*UnitOfWork](NewUnitOfWork)
		return sc
	}
}
