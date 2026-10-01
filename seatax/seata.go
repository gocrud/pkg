package seatax

import (
	"context"

	"github.com/gocrud/pkg/infra"
	"gorm.io/gorm"
)

type Seata struct {
	db *gorm.DB
}

func NewSeata(db *gorm.DB) *Seata {
	return &Seata{
		db: db,
	}
}

func (s *Seata) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(context.WithValue(ctx, infra.TxKey{}, s.db.WithContext(ctx)))
}
