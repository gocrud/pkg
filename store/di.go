package store

import (
	"fmt"

	"github.com/gocrud/kernel"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// AddGorm 把 *GormDB 注册为单例,依赖已注册的 *gorm.DB(AddDB 或
// seatax.AddGormDb 提供);无 Redis 的模块只 Extend 它即可。
//
//	kernel.New().Extend(store.AddDB(store.MySQL, dsn)).Extend(store.AddGorm)
func AddGorm(b *kernel.AppBuilder) {
	b.TryProvide[*GormDB](NewGormDB)
}

// DriverName 是数据库驱动名,取值见 MySQL / Postgres。
type DriverName string

const (
	// MySQL 使用 gorm.io/driver/mysql,DSN 形如 "user:pass@tcp(127.0.0.1:3306)/db?parseTime=true"。
	MySQL DriverName = "mysql"
	// Postgres 使用 gorm.io/driver/postgres,DSN 形如 "host=127.0.0.1 user=postgres dbname=db sslmode=disable"。
	Postgres DriverName = "postgres"
)

// AddDB 按 driver 与 dsn 打开 *gorm.DB 并注册为单例;需要 seata 代理连接时改用
// seatax.AddGormDb。
//
//	kernel.New().Extend(store.AddDB(store.MySQL, dsn))
func AddDB(driver DriverName, dsn string) kernel.Extension {
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

// AddRedis 按 cfg 注册 redis.UniversalClient 与 *Redis,不依赖其它组件。
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
		b.TryProvide[*Redis](NewRedis)
	}
}
