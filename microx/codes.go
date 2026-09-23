package microx

import (
	"net/http"

	"google.golang.org/grpc/codes"
)

// 业务错误分类对应的 HTTP 状态码。go-micro 结构化错误的 Code 字段采用 HTTP
// 风格,server/grpc 会据此自动映射为 gRPC status code,业务方只需关心 errorx
// 错误码即可。
const (
	HTTPOK            = int32(http.StatusOK)                  // 200 → gRPC OK
	HTTPBadRequest    = int32(http.StatusBadRequest)          // 400 → gRPC InvalidArgument
	HTTPUnauthorized  = int32(http.StatusUnauthorized)        // 401 → gRPC Unauthenticated
	HTTPInternalError = int32(http.StatusInternalServerError) // 500 → gRPC Internal
)

// HTTPCodeToGRPC 将 HTTP 风格错误码翻译为 gRPC status code。
// 映射与 go-micro server/grpc 的 microError 表保持一致。
func HTTPCodeToGRPC(code int32) codes.Code {
	switch code {
	case http.StatusOK:
		return codes.OK
	case http.StatusBadRequest:
		return codes.InvalidArgument
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	case http.StatusForbidden:
		return codes.PermissionDenied
	case http.StatusNotFound:
		return codes.NotFound
	case http.StatusRequestTimeout:
		return codes.DeadlineExceeded
	case http.StatusConflict:
		return codes.AlreadyExists
	case http.StatusPreconditionFailed:
		return codes.FailedPrecondition
	case http.StatusTooManyRequests:
		return codes.ResourceExhausted
	case http.StatusInternalServerError:
		return codes.Internal
	case http.StatusNotImplemented:
		return codes.Unimplemented
	case http.StatusServiceUnavailable:
		return codes.Unavailable
	default:
		return codes.Unknown
	}
}

// GRPCCodeToHTTP 将 gRPC status code 翻译为 HTTP 风格错误码。
// 映射与 go-micro client/grpc 的 microStatusFromGrpcCode 表保持一致。
func GRPCCodeToHTTP(code codes.Code) int32 {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.NotFound:
		return http.StatusNotFound
	case codes.DeadlineExceeded:
		return http.StatusRequestTimeout
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.FailedPrecondition:
		return http.StatusPreconditionFailed
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
