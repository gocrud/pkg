package store

import (
	"context"

	"github.com/gocrud/pkg/gormctx"
	"gorm.io/gorm"
)

// GormDB 持有根 *gorm.DB,按 ctx 解析本次读写实际使用的连接。
type GormDB struct {
	db *gorm.DB
}

// NewGormDB 的 db 不能为空。
func NewGormDB(db *gorm.DB) *GormDB {
	if db == nil {
		panic("store: NewGormDB 的 db 不能为空")
	}
	return &GormDB{db: db}
}

// WithContext 返回绑定 ctx 的 *gorm.DB,优先使用 ctx 内已注入的事务连接。
func (d *GormDB) WithContext(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(gormctx.TxKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return d.db.WithContext(ctx)
}
