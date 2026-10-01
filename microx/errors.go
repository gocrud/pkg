package microx

import (
	"net/http"
)

// 业务错误分类对应的 HTTP 状态码。go-micro 结构化错误的 Code 字段采用 HTTP
// 风格,server/grpc 会据此自动映射为 gRPC status code,业务方只需关心业务
// 错误码(见下方常量与 errorx.Define)即可。
const (
	HTTPBadRequest    = int32(http.StatusBadRequest)          // 400 → gRPC InvalidArgument
	HTTPUnauthorized  = int32(http.StatusUnauthorized)        // 401 → gRPC Unauthenticated
	HTTPInternalError = int32(http.StatusInternalServerError) // 500 → gRPC Internal
)

// 本包协议层错误码,写入结构化错误的 Reason 字段(经由 go-micro 透传)。
// errorx 只负责错误码定义与消息渲染,不再内置具体协议码,由各出口包自行定义;
// 业务自定义错误码请使用 errorx.Define。
const (
	// ErrParam 参数校验失败。
	ErrParam = "ERR_PARAM"
	// ErrUnauthorized 未认证或凭证失效。
	ErrUnauthorized = "ERR_UNAUTH"
	// ErrInternal 内部错误。出口层据此隐藏细节,仅返回兑底提示。
	ErrInternal = "ERR_SYS"
)
