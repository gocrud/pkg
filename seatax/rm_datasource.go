package seatax

import (
	"database/sql"

	"github.com/gocrud/pkg/errorx"
	seatasql "seata.apache.org/seata-go/v2/pkg/datasource/sql"
)

// seata RM 数据源驱动名,与 SDK 注册名一致。须先调用 Init 完成客户端初始化,
// SDK 才会注册这些驱动。
const (
	DriverATMySQL    = seatasql.SeataATMySQLDriver
	DriverATPostgres = seatasql.SeataATPostgresDriver
	DriverXAMySQL    = seatasql.SeataXAMySQLDriver
	DriverXAPostgres = seatasql.SeataXAPostgresDriver
)

// OpenATMySQL 打开 AT 模式 MySQL 数据源,返回 *sql.DB,可直接交给 gorm 使用。
func OpenATMySQL(dsn string) (*sql.DB, error) {
	return openSeataDB(DriverATMySQL, dsn)
}

// OpenATPostgres 打开 AT 模式 PostgreSQL 数据源。
func OpenATPostgres(dsn string) (*sql.DB, error) {
	return openSeataDB(DriverATPostgres, dsn)
}

// OpenXAMySQL 打开 XA 模式 MySQL 数据源。
func OpenXAMySQL(dsn string) (*sql.DB, error) {
	return openSeataDB(DriverXAMySQL, dsn)
}

// OpenXAPostgres 打开 XA 模式 PostgreSQL 数据源。
func OpenXAPostgres(dsn string) (*sql.DB, error) {
	return openSeataDB(DriverXAPostgres, dsn)
}

func openSeataDB(driverName, dsn string) (*sql.DB, error) {
	if dsn == "" {
		return nil, errorx.E(errorx.ErrParam, "数据源 DSN 不能为空")
	}
	if !driverRegistered(driverName) {
		return nil, errorx.E(CodeConfig, "seata 数据源驱动未注册,请先调用 seatax.Init")
	}
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, errorx.E(CodeConfig, "打开 seata 数据源失败", err)
	}
	return db, nil
}

func driverRegistered(name string) bool {
	for _, registered := range sql.Drivers() {
		if registered == name {
			return true
		}
	}
	return false
}
