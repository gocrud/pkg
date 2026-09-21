package grpcx

import (
	"context"

	"google.golang.org/grpc"
)

type verifierSniffer interface {
	Validate() error
}

func UnaryServerValidationInterceptor(skipMethods ...string) grpc.UnaryServerInterceptor {
	skipped := make(map[string]bool, len(skipMethods))
	for _, method := range skipMethods {
		skipped[method] = true
	}
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if skipped[info.FullMethod] {
			return handler(ctx, req)
		}
		if verifier, ok := req.(verifierSniffer); ok {
			if err := verifier.Validate(); err != nil {
				return nil, HandleServerError(ctx, err)
			}
		}
		return handler(ctx, req)
	}
}