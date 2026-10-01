package seatax

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// UndoLog 是 Seata AT 模式依赖的业务库 undo_log 表实体。
//
// 它与 seatax/undo_log.sql（MySQL）、seatax/undo_log_pg.sql（PostgreSQL）两份脚本
// 描述的表结构一致：字符串列统一使用 text，二进制列使用 bytea/longblob，
// 主键自增由 GORM 按方言分别渲染为 bigserial / bigint auto_increment。
//
// 说明：time.Time 在 PostgreSQL 下默认渲染为 timestamptz，如需与脚本的
// 无时区 timestamp 完全一致，可自行把 tag 改成 type:timestamp（MySQL 侧则不建议，
// 因为 MySQL 的 timestamp 有 2038 上限，脚本用的是 datetime）。
//
// 列类型按方言渲染：id → bigserial（PostgreSQL）/ bigint AUTO_INCREMENT（MySQL），
// xid、context、ext → text，rollback_info → bytea / longblob，
// log_status → integer / int。MigrateUndoLog 只负责建表、不改动既有表，
// 因此列类型与脚本之间的等价差异（如 datetime(3) 对 datetime）不会触发 ALTER。
type UndoLog struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	BranchID     int64     `gorm:"column:branch_id;not null;uniqueIndex:ux_undo_log,priority:2;comment:分支事务ID"`
	Xid          string    `gorm:"column:xid;type:text;not null;uniqueIndex:ux_undo_log,priority:1;comment:全局事务ID"`
	Context      string    `gorm:"column:context;type:text;not null;comment:回滚上下文"`
	RollbackInfo []byte    `gorm:"column:rollback_info;not null;comment:回滚数据(序列化后的前/后镜像)"`
	LogStatus    int32     `gorm:"column:log_status;not null;comment:状态:0正常,1全局事务已完成"`
	LogCreated   time.Time `gorm:"column:log_created;not null;index:ix_log_created;comment:创建时间"`
	LogModified  time.Time `gorm:"column:log_modified;not null;comment:修改时间"`
	Ext          string    `gorm:"column:ext;type:text;comment:扩展信息"`
}

// TableName 固定表名为 undo_log（GORM 默认会推导为 undo_logs）。
func (UndoLog) TableName() string { return "undo_log" }

// MigrateUndoLog 保证业务库中存在 undo_log 表：表不存在时按 UndoLog 实体建表
// （含唯一索引 ux_undo_log 与索引 ix_log_created），表已存在时不做任何 DDL，
// 因此它不会删除数据，也不会对用 undo_log.sql / undo_log_pg.sql 建好的表做
// 等价但代价高的类型改写（如 int → bigint、datetime → datetime(3)）。
//
// 如需增量对齐实体与既有表结构，可自行调用 db.AutoMigrate(&seatax.UndoLog{})。
//
// db 为业务库的 GORM 连接（MySQL/PostgreSQL 均可，由 GORM 方言自动适配），
// 例如 seatax.GetGormDb 的返回值。db 为 nil 时返回错误。
func MigrateUndoLog(db *gorm.DB) error {
	if db == nil {
		return errors.New("seatax: nil *gorm.DB passed to MigrateUndoLog")
	}
	m := db.Migrator()
	if m.HasTable(&UndoLog{}) {
		return nil
	}
	if err := m.CreateTable(&UndoLog{}); err != nil {
		return fmt.Errorf("seatax: create undo_log table: %w", err)
	}
	return nil
}
