package store

import (
	"fmt"

	"github.com/gocrud/kernel"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// AddStore 把 Store 注册为单例,依赖已注册的 *gorm.DB(由 AddGormDB 或
// seatax.AddGormDb 注册)与 redis.UniversalClient(由 AddRedis 注册)。
//
// 遵循 kernel 的扩展约定:只做注册、无返回值,内部使用 TryProvide 保证幂等,
// 既可单独 Extend,也可放进 []kernel.Extension 批量装配:
//
//	kernel.New().
//	    Extend(seatax.AddGormDb(seatax.ATMySQL, dsn)).
//	    Extend(store.AddRedis(store.RedisConfig{Addr: "127.0.0.1:6379"})).
//	    Extend(store.AddStore).
//	    Extend(uow.AddUnitOfWork)
func AddStore(b *kernel.AppBuilder) {
	b.TryProvide[Store](NewStore)
}

// DriverName 是数据库驱动名,取值见 MySQL / Postgres 常量。
type DriverName string

const (
	// MySQL 使用 gorm.io/driver/mysql,DSN 形如 "user:pass@tcp(127.0.0.1:3306)/db?parseTime=true"。
	MySQL DriverName = "mysql"
	// Postgres 使用 gorm.io/driver/postgres,DSN 形如 "host=127.0.0.1 user=postgres dbname=db sslmode=disable"。
	Postgres DriverName = "postgres"
)

// AddGormDB 返回一个 kernel 扩展,按驱动名与 DSN 打开 *gorm.DB 并注册为单例,
// 供 AddStore 自动注入。目前支持 MySQL 与 Postgres;需要 seata 代理连接时改用
// seatax.AddGormDb。
//
//	kernel.New().Extend(store.AddGormDB(store.MySQL, dsn))
func AddGormDB(driver DriverName, dsn string) kernel.Extension {
	return func(b *kernel.AppBuilder) {
		b.TryProvide[*gorm.DB](func() (*gorm.DB, error) {
			switch driver {
			case MySQL:
				return gorm.Open(mysql.Open(dsn), &gorm.Config{})
			case Postgres:
				return gorm.Open(postgres.Open(dsn), &gorm.Config{})
			default:
				return nil, fmt.Errorf("store: 不支持的驱动 %q", driver)
			}
		})
	}
}

// RedisConfig 是 Redis 客户端的最小连接配置。
type RedisConfig struct {
	// Addr 形如 "127.0.0.1:6379"。
	Addr string
	// Password 为空表示无密码。
	Password string
	// DB 选择逻辑库,默认 0。
	DB int
}

// AddRedis 返回一个 kernel 扩展,按 cfg 注册 redis.UniversalClient 单例,供
// AddStore 自动注入。需要参数时用返回闭包的形式:
//
//	kernel.New().Extend(store.AddRedis(store.RedisConfig{Addr: "127.0.0.1:6379"}))
func AddRedis(cfg RedisConfig) kernel.Extension {
	return func(b *kernel.AppBuilder) {
		b.TryProvide[redis.UniversalClient](func() (redis.UniversalClient, error) {
			return redis.NewUniversalClient(&redis.UniversalOptions{
				Addrs:    []string{cfg.Addr},
				Password: cfg.Password,
				DB:       cfg.DB,
			}), nil
		})
	}
}
