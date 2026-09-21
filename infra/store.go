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
	return TxOrDB(ctx, store.db, TxOptional)
}

func (store *Store) ForUpdate(ctx context.Context) *gorm.DB {
	query := TxOrDB(ctx, store.db, TxRequired)
	if query.Error != nil {
		return query
	}
	return query.Clauses(clause.Locking{Strength: "UPDATE"})
}

func AddStore() ioc.ServiceCollectionExtension {
	return func(sc *ioc.ServiceCollection) *ioc.ServiceCollection {
		sc.TryAddSingleton[*Store](NewStore)
		return sc
	}
}
