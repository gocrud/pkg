package seatax

import (
	"database/sql"
	"slices"

	"github.com/gocrud/pkg/errorx"
)

// OpenDataSource 打开 seata 代理数据源,返回 *sql.DB,可直接执行 SQL 或交给
// WrapGorm 包装成 gorm。mode 取 ModeAT / ModeXA,dbType 取 DBTypeMySQL /
// DBTypePostgres。须先调用 Init 完成客户端初始化,SDK 才会注册这些驱动;
// 驱动未注册时返回 SEATA_CONFIG 错误。
func OpenDataSource(mode Mode, dbType DBType, dsn string) (*sql.DB, error) {
	spec, err := dbSpecFor(dbType)
	if err != nil {
		return nil, err
	}
	return openSeataDB(spec.driver(mode), dsn)
}

func openSeataDB(driverName, dsn string) (*sql.DB, error) {
	if dsn == "" {
		return nil, newBizErr(errorx.ErrParam, "数据源 DSN 不能为空")
	}
	if !driverRegistered(driverName) {
		return nil, newBizErr(CodeConfig, "seata 数据源驱动未注册,请先调用 seatax.Init")
	}
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, newBizErr(CodeConfig, "打开 seata 数据源失败", err)
	}
	return db, nil
}

func driverRegistered(name string) bool {
	return slices.Contains(sql.Drivers(), name)
}
