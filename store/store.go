package store

import (
	"context"

	"github.com/gocrud/pkg/gormctx"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Store 是数据访问的聚合入口,把关系库(gorm)与缓存(redis)等基础设施客户端
// 收拢在一起,业务 repo 统一依赖本接口。
//
// 事务语义只属于关系库:
//   - 无事务时,GormDB(ctx) 返回根连接;
//   - 处于 uow.UnitOfWork.Do(本地事务)或 seatax.Seata.Do(分布式事务)内时,
//     返回当前事务连接,使 repo 的每次读写自动落入同一事务。
//
// Redis 等其它资源无 ACID 事务,通过 Redis() 直取客户端即可。
type Store interface {
	// GormDB 返回绑定 ctx 的 *gorm.DB。若 ctx 内已注入事务连接,则返回该事务
	// 连接,否则返回根连接。
	GormDB(ctx context.Context) *gorm.DB
	// Redis 返回 Redis 客户端(无状态、可复用,无需按 ctx 取)。未配置时返回 nil。
	Redis() redis.UniversalClient
}

// storeImpl 是 Store 的默认实现,包装根连接与 Redis 客户端。
type storeImpl struct {
	db  *gorm.DB
	rdb redis.UniversalClient
}

// GormDB 实现 Store。
func (s *storeImpl) GormDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(gormctx.TxKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return s.db.WithContext(ctx)
}

// Redis 实现 Store。
func (s *storeImpl) Redis() redis.UniversalClient {
	return s.rdb
}

// NewStore 以根连接 db 与 Redis 客户端 rdb 构造 Store。db 不能为空;rdb 可为
// nil(仅使用关系库时)。
func NewStore(db *gorm.DB, rdb redis.UniversalClient) Store {
	if db == nil {
		panic("store: NewStore 的 db 不能为空")
	}
	return &storeImpl{db: db, rdb: rdb}
}
