package microx

import (
	"context"

	"go-micro.dev/v6/server"
)

// verifierSniffer 是支持校验的请求体接口,与 grpcx 的约定一致。
type verifierSniffer interface {
	Validate() error
}

// ValidationHandlerWrapper 返回一个 server.HandlerWrapper:在调用业务 handler
// 前,若请求体实现了 Validate 方法则先执行校验,校验失败统一经 ToMicroError
// 翻译后返回。skipEndpoints 可指定跳过校验的端点名(如 "Health.Check")。
func ValidationHandlerWrapper(skipEndpoints ...string) server.HandlerWrapper {
	skipped := make(map[string]bool, len(skipEndpoints))
	for _, endpoint := range skipEndpoints {
		skipped[endpoint] = true
	}
	return func(next server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			if skipped[req.Endpoint()] {
				return next(ctx, req, rsp)
			}
			if verifier, ok := req.Body().(verifierSniffer); ok {
				if err := verifier.Validate(); err != nil {
					return ToMicroError(err)
				}
			}
			return next(ctx, req, rsp)
		}
	}
}
