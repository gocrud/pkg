package seatax

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/gocrud/ioc"
	"github.com/gocrud/pkg/errorx"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Mode 区分 RM 事务模式。
type Mode int

const (
	// ModeAT AT 模式,依赖 undo_log 表。
	ModeAT Mode = iota
	// ModeXA XA 模式,要求数据库支持 XA 协议(MySQL 8+,PostgreSQL 需相应插件)。
	ModeXA
)

// OpenGorm 打开 seata 数据源并返回 gorm.DB。dialect 支持
// mysql / postgres(或 postgresql、pgsql)。底层连接使用 seata 代理驱动,
// 表结构元数据缓存、undo 日志等均由 SDK 处理。
func OpenGorm(mode Mode, dialect, dsn string) (*gorm.DB, error) {
	driverName, err := driverFor(mode, dialect)
	if err != nil {
		return nil, err
	}
	conn, err := openSeataDB(driverName, dsn)
	if err != nil {
		return nil, err
	}
	dialector, err := gormDialector(dialect, conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		_ = conn.Close()
		return nil, errorx.E(CodeConfig, "初始化 seata gorm 失败", err)
	}
	return db, nil
}

// OpenATGorm 打开 AT 模式 gorm,等价于 OpenGorm(ModeAT, dialect, dsn)。
func OpenATGorm(dialect, dsn string) (*gorm.DB, error) {
	return OpenGorm(ModeAT, dialect, dsn)
}

// OpenXAGorm 打开 XA 模式 gorm,等价于 OpenGorm(ModeXA, dialect, dsn)。
func OpenXAGorm(dialect, dsn string) (*gorm.DB, error) {
	return OpenGorm(ModeXA, dialect, dsn)
}

// driverFor 将方言与事务模式映射为 seata 代理驱动名。
func driverFor(mode Mode, dialect string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "mysql":
		if mode == ModeXA {
			return DriverXAMySQL, nil
		}
		return DriverATMySQL, nil
	case "postgres", "postgresql", "pgsql":
		if mode == ModeXA {
			return DriverXAPostgres, nil
		}
		return DriverATPostgres, nil
	default:
		return "", errorx.E(errorx.ErrParam, fmt.Sprintf("不支持的数据方言: %q", dialect))
	}
}

func gormDialector(dialect string, conn *sql.DB) (gorm.Dialector, error) {
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "mysql":
		return mysql.New(mysql.Config{Conn: conn}), nil
	case "postgres", "postgresql", "pgsql":
		return postgres.New(postgres.Config{Conn: conn}), nil
	default:
		return nil, errorx.E(errorx.ErrParam, fmt.Sprintf("不支持的数据方言: %q", dialect))
	}
}

// AddDatabase 返回一个 ioc 扩展,把 seata 数据源注册为 *gorm.DB 单例,
// 与 infra.AddDatabase 的注册约定一致。driver 可传:
//   - seata 驱动名:seata-at-mysql / seata-at-postgres / seata-xa-mysql / seata-xa-postgres
//   - 方言简写:mysql / postgres(等价于 AT 模式)
func AddDatabase(dsn string, driver ...string) ioc.ServiceCollectionExtension {
	return func(sc *ioc.ServiceCollection) *ioc.ServiceCollection {
		sc.TryAddSingleton[*gorm.DB](func() (*gorm.DB, error) {
			mode, dialect, err := parseSeataDriver(firstDriver(driver...))
			if err != nil {
				return nil, err
			}
			return OpenGorm(mode, dialect, dsn)
		})
		return sc
	}
}

func firstDriver(driver ...string) string {
	if len(driver) > 1 {
		return ""
	}
	if len(driver) == 1 {
		return driver[0]
	}
	return "mysql"
}

func parseSeataDriver(name string) (Mode, string, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case DriverATMySQL, "mysql":
		return ModeAT, "mysql", nil
	case DriverATPostgres, "postgres", "postgresql", "pgsql":
		return ModeAT, "postgres", nil
	case DriverXAMySQL:
		return ModeXA, "mysql", nil
	case DriverXAPostgres:
		return ModeXA, "postgres", nil
	default:
		return ModeAT, "", errorx.E(errorx.ErrParam, fmt.Sprintf("不支持的 seata 数据源驱动: %q", name))
	}
}
