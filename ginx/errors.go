package ginx

// 本包协议层错误码,写入统一响应 Result.Code。
// errorx 只负责错误码定义与消息渲染,不再内置具体协议码,由各出口包自行定义;
// 业务自定义错误码请使用 errorx.Define。
const (
	// ErrOK 成功。
	ErrOK = "SUCCESS"
	// ErrParam 参数错误。
	ErrParam = "ERR_PARAM"
	// ErrInternal 内部错误。中间件据此隐藏细节,仅返回兑底提示。
	ErrInternal = "ERR_SYS"
	// ErrForbidden 无权限访问。
	ErrForbidden = "ERR_FORBIDDEN"
)
