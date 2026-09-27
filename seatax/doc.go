// Package seatax 封装 seata.apache.org/seata-go/v2,提供 TM(全局事务)与
// RM(AT / XA 数据源、TCC)能力,并针对 gRPC、HTTP(gin)、go-micro 三种协议
// 提供 XID 传播组件。
//
// # 初始化
//
// 使用前必须先 Init(配置文件路径)或 InitFromConf(内嵌配置);代理数据库驱动在
// 初始化成功后才注册,数据源须在其后打开。
//
// # 常用入口
//
//   - TM:WithGlobalTx(ctx, name, fn, opts...) 按回调结果提交或回滚,业务错误原样透传;
//     GetXID(ctx) 读取当前全局事务 XID。
//   - RM:OpenDataSource(mode, dbType, dsn)、WrapGorm(dbType, conn)、
//     OpenGorm(mode, dbType, dsn);AddDatabase(dsn, opts...) 为 IoC 注册。
//   - TCC:NewTCCProxy(service) 注册资源;BusinessActionContext 为二阶段上下文。
//   - XID 传播:GinTransactionMiddleware、ServerTransactionInterceptor、
//     ClientTransactionInterceptor、MicroServerTransactionWrapper、
//     MicroClientTransactionWrapper。
//
// 所有 seata 内部错误统一转换为 errorx 业务错误(SEATA_* 错误码),业务错误原样
// 透传,与 grpcx / ginx / microx 的统一错误处理链路兼容。详见本包 README。
package seatax
