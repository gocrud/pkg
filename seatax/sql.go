package seatax

import (
	"database/sql"
	"fmt"

	sql2 "seata.apache.org/seata-go/v2/pkg/datasource/sql"
)

type DriverName string

const (
	ATMySQL    DriverName = sql2.SeataATMySQLDriver
	ATPostgres DriverName = sql2.SeataATPostgresDriver
	XAMySQL    DriverName = sql2.SeataXAMySQLDriver
	XAPostgres DriverName = sql2.SeataXAPostgresDriver
)

// GetSqlDb opens a database connection using the specified driver name and data source name. It returns the sql.DB instance and any error encountered.
func GetSqlDb(driverName DriverName, dataSourceName string) (*sql.DB, error) {
	// driverName 有效性检查
	switch driverName {
	case ATMySQL, ATPostgres, XAMySQL, XAPostgres:
		// 有效的 driverName
	default:
		return nil, fmt.Errorf("unsupported driver: %s", driverName)
	}
	return sql.Open(string(driverName), dataSourceName)
}
