package grpcx

// 本包协议层错误码,写入 trailer 的 x-biz-code。
// errorx 只负责错误码定义与消息渲染,不再内置具体协议码,由各出口包自行定义;
// 业务自定义错误码请使用 errorx.Define。
const (
	// ErrParam 参数校验失败。
	ErrParam = "ERR_PARAM"
	// ErrInternal 内部错误。出口层据此隐藏细节,仅返回兑底提示。
	ErrInternal = "ERR_SYS"
)
