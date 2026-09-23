package seatax

import (
	"fmt"
	"time"

	"github.com/gocrud/pkg/errorx"
	"gorm.io/gorm"
)

// undoLogMySQL 对应 seata 0.3.0+ 的 MySQL undo_log 表结构。
// 注意唯一索引 ux_undo_log(xid, branch_id) 通过 gorm tag 声明。
type undoLogMySQL struct {
	ID           int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement;not null"`
	BranchID     int64     `gorm:"column:branch_id;type:bigint;not null;uniqueIndex:ux_undo_log,priority:2"`
	XID          string    `gorm:"column:xid;type:varchar(100);not null;uniqueIndex:ux_undo_log,priority:1"`
	Context      string    `gorm:"column:context;type:varchar(128);not null"`
	RollbackInfo []byte    `gorm:"column:rollback_info;type:longblob;not null"`
	LogStatus    int       `gorm:"column:log_status;type:int(11);not null"`
	LogCreated   time.Time `gorm:"column:log_created;type:datetime;not null"`
	LogModified  time.Time `gorm:"column:log_modified;type:datetime;not null"`
	Ext          *string   `gorm:"column:ext;type:varchar(100)"`
}

func (undoLogMySQL) TableName() string { return "undo_log" }

// undoLogPostgres 对应 PostgreSQL 的 undo_log 表结构,字符串列采用 text,
// 二进制列采用 bytea,时间列与 seata 官方 DDL 一致使用 timestamp(0)。
type undoLogPostgres struct {
	ID           int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement;not null"`
	BranchID     int64     `gorm:"column:branch_id;type:bigint;not null;uniqueIndex:ux_undo_log,priority:2"`
	XID          string    `gorm:"column:xid;type:text;not null;uniqueIndex:ux_undo_log,priority:1"`
	Context      string    `gorm:"column:context;type:text;not null"`
	RollbackInfo []byte    `gorm:"column:rollback_info;type:bytea;not null"`
	LogStatus    int       `gorm:"column:log_status;type:integer;not null"`
	LogCreated   time.Time `gorm:"column:log_created;type:timestamp(0);not null"`
	LogModified  time.Time `gorm:"column:log_modified;type:timestamp(0);not null"`
	Ext          *string   `gorm:"column:ext;type:text"`
}

func (undoLogPostgres) TableName() string { return "undo_log" }

// InitUndoLogMySQL 使用 gorm 结构体建表方式初始化 AT 模式所需的 MySQL
// undo_log 表(含唯一索引 ux_undo_log)。建议传入指向业务库的普通 *gorm.DB,
// 而非 seata 代理连接;重复调用幂等(AutoMigrate 只补缺失的列/索引)。
func InitUndoLogMySQL(db *gorm.DB) error {
	return initUndoLog(db, "MySQL", &undoLogMySQL{})
}

// InitUndoLogPostgres 使用 gorm 结构体建表方式初始化 AT 模式所需的 PostgreSQL
// undo_log 表(含唯一索引 ux_undo_log)。建议传入指向业务库的普通 *gorm.DB,
// 而非 seata 代理连接;重复调用幂等(AutoMigrate 只补缺失的列/索引)。
func InitUndoLogPostgres(db *gorm.DB) error {
	return initUndoLog(db, "PostgreSQL", &undoLogPostgres{})
}

func initUndoLog(db *gorm.DB, dialect string, model any) error {
	if db == nil {
		return errorx.E(errorx.ErrParam, "gorm.DB 不能为空")
	}
	if err := db.AutoMigrate(model); err != nil {
		return errorx.E(CodeConfig, fmt.Sprintf("初始化 %s undo_log 表失败", dialect), err)
	}
	return nil
}
