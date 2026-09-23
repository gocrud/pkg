package seatax

// seatax 业务错误码。所有由 seata SDK / TC 交互产生的错误,都会转换为携带
// 下列错误码的 errorx.BizError,业务端可沿用 errorx / grpcx / httpx / microx
// 的统一错误处理链路。
const (
	// CodeBegin 全局事务开启失败(TC 不可达、超时等)。
	CodeBegin = "SEATA_BEGIN"
	// CodeCommit 全局事务提交失败。
	CodeCommit = "SEATA_COMMIT"
	// CodeRollback 全局事务回滚失败。
	CodeRollback = "SEATA_ROLLBACK"
	// CodeRegister 分支事务资源注册失败(如 TCC 资源注册)。
	CodeRegister = "SEATA_REGISTER"
	// CodeXIDMissing 严格模式下缺少全局事务 XID。
	CodeXIDMissing = "SEATA_XID_MISSING"
	// CodeConfig seata 客户端初始化 / 配置错误。
	CodeConfig = "SEATA_CONFIG"
	// CodeInternal 未识别的 seata 内部错误。
	CodeInternal = "SEATA_INTERNAL"
)
