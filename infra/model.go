package infra

import "gorm.io/plugin/soft_delete"

type BaseModel struct {
	ID        int64                 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	CreatedAt int64                 `gorm:"column:created_at;autoCreateTime;comment:创建时间(秒级时间戳)"`
	UpdatedAt int64                 `gorm:"column:updated_at;autoUpdateTime;comment:更新时间(秒级时间戳)"`
	DeletedAt soft_delete.DeletedAt `gorm:"column:deleted_at;softDelete;default:0;comment:删除时间(0未删除)"`
}
