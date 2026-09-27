package seatax

import (
	"database/sql"
	"fmt"

	"github.com/gocrud/ioc"
	"github.com/gocrud/pkg/errorx"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	seatasql "seata.apache.org/seata-go/v2/pkg/datasource/sql"
)

// dbSpec 描述一种数据库在 seata 中的 AT/XA 代理驱动名与 gorm dialector 构造方式。
type dbSpec struct {
	atDriver  string
	xaDriver  string
	dialector func(conn *sql.DB) gorm.Dialector
}

// driver 按事务模式返回 seata 代理驱动名。
func (s dbSpec) driver(mode Mode) string {
	if mode == ModeXA {
		return s.xaDriver
	}
	return s.atDriver
}

// dbSpecs 汇总每种数据库类型对应的驱动与 dialector,是驱动名与 dialector
// 的单一数据源,避免两处 switch 重复。
var dbSpecs = map[DBType]dbSpec{
	DBTypeMySQL: {
		atDriver:  seatasql.SeataATMySQLDriver,
		xaDriver:  seatasql.SeataXAMySQLDriver,
		dialector: func(conn *sql.DB) gorm.Dialector { return mysql.New(mysql.Config{Conn: conn}) },
	},
	DBTypePostgres: {
		atDriver:  seatasql.SeataATPostgresDriver,
		xaDriver:  seatasql.SeataXAPostgresDriver,
		dialector: func(conn *sql.DB) gorm.Dialector { return postgres.New(postgres.Config{Conn: conn}) },
	},
}

// dbSpecFor 按数据库类型查找 dbSpec,不支持的数据库类型返回 ERR_PARAM。
func dbSpecFor(dbType DBType) (dbSpec, error) {
	spec, ok := dbSpecs[dbType]
	if !ok {
		return dbSpec{}, newBizErr(errorx.ErrParam, fmt.Sprintf("不支持的数据库类型: %q", dbType))
	}
	return spec, nil
}

// WrapGorm 使用已打开的 seata 数据源连接创建 *gorm.DB,与官方示例
// gorm.Open(mysql.New(mysql.Config{Conn: sqlDB}), &gorm.Config{}) 一致。
// dbType 支持 DBTypeMySQL / DBTypePostgres,conn 应来自 OpenDataSource
// (seata 代理驱动连接),表结构元数据缓存、undo 日志等均由 SDK 处理。
func WrapGorm(dbType DBType, conn *sql.DB) (*gorm.DB, error) {
	if conn == nil {
		return nil, newBizErr(errorx.ErrParam, "数据源连接不能为空")
	}
	spec, err := dbSpecFor(dbType)
	if err != nil {
		return nil, err
	}
	db, err := gorm.Open(spec.dialector(conn), &gorm.Config{})
	if err != nil {
		return nil, newBizErr(CodeConfig, "初始化 seata gorm 失败", err)
	}
	return db, nil
}

// OpenGorm 打开 seata 代理数据源并包装为 *gorm.DB,是 OpenDataSource +
// WrapGorm 的组合便捷方法。
func OpenGorm(mode Mode, dbType DBType, dsn string) (*gorm.DB, error) {
	conn, err := OpenDataSource(mode, dbType, dsn)
	if err != nil {
		return nil, err
	}
	db, err := WrapGorm(dbType, conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return db, nil
}

// DBOption 数据源注册选项。
type DBOption func(*databaseOptions)

type databaseOptions struct {
	mode   Mode
	dbType DBType
}

// WithMode 设置事务模式,默认 ModeAT。
func WithMode(mode Mode) DBOption {
	return func(o *databaseOptions) {
		o.mode = mode
	}
}

// WithDBType 设置数据库类型,默认 DBTypeMySQL。
func WithDBType(dbType DBType) DBOption {
	return func(o *databaseOptions) {
		o.dbType = dbType
	}
}

// AddDatabase 返回一个 ioc 扩展,把 seata 数据源注册为 *gorm.DB 单例,
// 与 infra.AddDatabase 的注册约定一致。默认 AT 模式 + MySQL,可用
// WithMode / WithDBType 调整。
func AddDatabase(dsn string, opts ...DBOption) ioc.ServiceCollectionExtension {
	return func(sc *ioc.ServiceCollection) *ioc.ServiceCollection {
		sc.TryAddSingleton[*gorm.DB](func() (*gorm.DB, error) {
			o := &databaseOptions{mode: ModeAT, dbType: DBTypeMySQL}
			for _, opt := range opts {
				opt(o)
			}
			return OpenGorm(o.mode, o.dbType, dsn)
		})
		return sc
	}
}
