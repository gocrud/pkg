package microx

import (
	"net/http"
)

// 业务错误分类对应的 HTTP 状态码。go-micro 结构化错误的 Code 字段采用 HTTP
// 风格,server/grpc 会据此自动映射为 gRPC status code,业务方只需关心 errorx
// 错误码即可。
const (
	HTTPBadRequest    = int32(http.StatusBadRequest)          // 400 → gRPC InvalidArgument
	HTTPUnauthorized  = int32(http.StatusUnauthorized)        // 401 → gRPC Unauthenticated
	HTTPInternalError = int32(http.StatusInternalServerError) // 500 → gRPC Internal
)
