// Package seatax 封装 seata.apache.org/seata-go/v2,提供 TM(全局事务)与
// RM(AT / XA 数据源)能力,并为 gin、gRPC、go-micro 三种协议提供 XID 传播组件。
//
// 导入路径为 github.com/gocrud/pkg/seatax,包名与目录一致:
//
//	import "github.com/gocrud/pkg/seatax"
//
// # 初始化
//
// # 常用入口
//
//   - TM:WithGlobalTx(ctx, gc, fn) 按回调结果提交或回滚,gc.Name 必填;
//     GtxConfig / CallbackWithCtx 为 tm 包类型的别名。
//   - RM:GetSqlDb(driverName, dsn) 返回代理 *sql.DB;GetGormDb(driverName, dsn)
//     直接返回 *gorm.DB;driverName 取 ATMySQL / ATPostgres / XAMySQL / XAPostgres。
//   - 事务边界:NewSeata(db) 与 (*Seata).Do(ctx, fn),把代理 *gorm.DB 注入
//     gormctx.TxKey,使 store.GormDB.WithContext(ctx) 命中该连接。
//   - AT 表结构:UndoLog 实体与 MigrateUndoLog(db) 按 MySQL 特性建表
//     (表名固定为 undo_log,表不存在时才建,已存在则不做任何 DDL),
//     与 undo_log.sql 逐列一致;PostgreSQL 后续单独提供实体(undo_log_pg.sql)。
//   - XID 传播:GinTransactionMiddleware、GrpcServerTransactionInterceptor、GrpcClientTransactionInterceptor、
//     MicroTransactionHandlerWrapper、MicroTransactionCallWrapper。
//
// seatax 不转换错误:业务回调错误原样透传,与 grpcx / ginx / microx 的统一错误
// 处理链路兼容。详见本包 README。
package seatax
