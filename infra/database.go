package infra

import (
	"fmt"
	"strings"

	"github.com/gocrud/ioc"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func AddDatabase(dsn string, driver ...string) ioc.ServiceCollectionExtension {
	return func(sc *ioc.ServiceCollection) *ioc.ServiceCollection {
		sc.TryAddSingleton[*gorm.DB](func() (*gorm.DB, error) {
			dialector, err := databaseDialector(dsn, driver...)
			if err != nil {
				return nil, err
			}
			return gorm.Open(dialector, &gorm.Config{})
		})
		return sc
	}
}

func databaseDialector(dsn string, driver ...string) (gorm.Dialector, error) {
	if len(driver) > 1 {
		return nil, fmt.Errorf("expected at most one database driver")
	}
	name := "mysql"
	if len(driver) == 1 {
		name = strings.ToLower(strings.TrimSpace(driver[0]))
	}
	switch name {
	case "mysql":
		return mysql.Open(dsn), nil
	case "pgsql", "postgres", "postgresql":
		return postgres.Open(dsn), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %q", name)
	}
}
