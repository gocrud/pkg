package seatax

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// UndoLog 是 Seata AT 模式依赖的业务库 undo_log 表实体，列定义**完全按 MySQL 表达**，
// 与 seatax/undo_log.sql 逐列一致；PostgreSQL 不使用本实体（后续单独提供）。
//
// 字段 → MySQL 列的对应关系：
//
//	ID           int64                   → bigint auto_increment
//	BranchID     int64                   → bigint
//	Xid          string  size:100        → varchar(100)
//	Context      string  size:128        → varchar(128)
//	RollbackInfo []byte                  → longblob
//	LogStatus    int32                   → int
//	LogCreated   time.Time type:datetime → datetime
//	LogModified  time.Time type:datetime → datetime
//	Ext          string  size:100        → varchar(100)（可空）
//
// 几处 MySQL 特性：
//   - 字符串列用 size 而不是 type:text：xid 参与唯一索引 ux_undo_log，而 MySQL 的
//     TEXT 列不能直接作为索引键（ERROR 1170，需前缀长度），varchar 天然可索引。
//   - 时间列显式 type:datetime，渲染结果与脚本的 datetime 完全一致（无小数秒，
//     即 datetime(0)）；MySQL 会把写入的多余小数秒四舍五入，若需毫秒/微秒精度，
//     改成 type:datetime(6)（Seata 官方 DDL 用的是 DATETIME(6)）。
//   - comment: 是 MySQL 专有语法，建表时生成 COMMENT '...'。
//   - int32 / int64 渲染出的 int / bigint 与脚本的 int(11) / bigint(20) 等价
//     （整数显示宽度自 MySQL 8.0.17 起已废弃，故不写宽度）。
//   - 表属性 ENGINE / CHARSET 不能写在 tag 里，由 MigrateUndoLog 通过
//     gorm:table_options 下发（见 undoLogTableOptions）。
//
// MigrateUndoLog 只负责建表、不改动既有表，因此不会对脚本建好的表做任何 ALTER。
type UndoLog struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	BranchID     int64     `gorm:"column:branch_id;not null;uniqueIndex:ux_undo_log,priority:2;comment:分支事务ID"`
	Xid          string    `gorm:"column:xid;size:100;not null;uniqueIndex:ux_undo_log,priority:1;comment:全局事务ID"`
	Context      string    `gorm:"column:context;size:128;not null;comment:回滚上下文"`
	RollbackInfo []byte    `gorm:"column:rollback_info;not null;comment:回滚数据(序列化后的前/后镜像)"`
	LogStatus    int32     `gorm:"column:log_status;not null;comment:状态:0正常,1全局事务已完成"`
	LogCreated   time.Time `gorm:"column:log_created;type:datetime;not null;index:ix_log_created;comment:创建时间"`
	LogModified  time.Time `gorm:"column:log_modified;type:datetime;not null;comment:修改时间"`
	Ext          string    `gorm:"column:ext;size:100;comment:扩展信息"`
}

// TableName 固定表名为 undo_log（GORM 默认会推导为 undo_logs）。
func (UndoLog) TableName() string { return "undo_log" }

// undoLogTableOptions 是 undo_log 的 MySQL 表属性，与 undo_log.sql 保持一致。
// GORM 的 tag 里没有 engine / charset 选项，只能建表前用 gorm:table_options 下发，
// 其他方言会忽略该设置。
const undoLogTableOptions = "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4"

// MigrateUndoLog 保证 MySQL 业务库中存在 undo_log 表：表不存在时按 UndoLog 实体建表
// （InnoDB + utf8mb4，含唯一索引 ux_undo_log 与索引 ix_log_created），
// 表已存在时不做任何 DDL，因此既不会删数据，也不会对用 undo_log.sql 建好的表
// 做等价但代价高的类型改写。
//
// 如需增量对齐实体与既有表结构，可自行调用 db.AutoMigrate(&seatax.UndoLog{})。
//
// 本函数面向 MySQL（实体即 MySQL 专用）；PostgreSQL 的实体与迁移函数后续单独提供。
//
// db 为 MySQL 业务库的 GORM 连接，例如 seatax.GetGormDb(seatax.ATMySQL, dsn)
// 的返回值。db 为 nil 时返回错误。
func MigrateUndoLog(db *gorm.DB) error {
	if db == nil {
		return errors.New("seatax: nil *gorm.DB passed to MigrateUndoLog")
	}
	m := db.Set("gorm:table_options", undoLogTableOptions).Migrator()
	if m.HasTable(&UndoLog{}) {
		return nil
	}
	if err := m.CreateTable(&UndoLog{}); err != nil {
		return fmt.Errorf("seatax: create undo_log table: %w", err)
	}
	return nil
}
