// Package gormx 提供基于 gorm.io/gorm 的模型基类 BaseModel。
//
// 导入路径为 github.com/gocrud/pkg/gormx,包名与目录一致:
//
//	import "github.com/gocrud/pkg/gormx"
//
// # 常用入口
//
//   - BaseModel:含自增主键、秒级时间戳与软删除的模型基类。
//
// 数据访问(按 ctx 取库)与本地事务分别见 store 与 uow 包。
package gormx
