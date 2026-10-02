package seatax

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// GetGormDb opens a Gorm database connection using the specified driver name and data source name. It returns the gorm.DB instance and any error encountered.
func GetGormDb(driverName DriverName, dataSourceName string) (*gorm.DB, error) {
	dialector, err := getDialector(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	return gorm.Open(dialector, &gorm.Config{})
}

func getDialector(driverName DriverName, dataSourceName string) (gorm.Dialector, error) {
	sqldb, err := GetSqlDb(driverName, dataSourceName)
	switch driverName {
	case ATMySQL, XAMySQL:
		if err != nil {
			return nil, err
		}
		return mysql.New(mysql.Config{Conn: sqldb}), nil
	default:
		return nil, fmt.Errorf("unsupported driver: %s", driverName)
	}
}
