package infra

import (
	"context"

	"github.com/gocrud/ioc"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (store *Store) WithContext(ctx context.Context) *gorm.DB {
	db := store.db
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok && tx != nil {
		db = tx
	}
	return db.Session(&gorm.Session{NewDB: true, Context: ctx})
}

func (store *Store) ForUpdate(ctx context.Context) *gorm.DB {
	query := store.WithContext(ctx)
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok && tx != nil {
		return query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return query
}

func AddStore() ioc.ServiceCollectionExtension {
	return func(sc *ioc.ServiceCollection) *ioc.ServiceCollection {
		sc.TryAddSingleton[*Store](NewStore)
		return sc
	}
}
