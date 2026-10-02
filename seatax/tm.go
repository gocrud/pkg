package seatax

import (
	"context"

	"seata.apache.org/seata-go/v2/pkg/tm"
)

type GtxConfig = tm.GtxConfig
type CallbackWithCtx = tm.CallbackWithCtx

// WithGlobalTx wraps the tm.WithGlobalTx function, providing a convenient way to execute a business function within a global transaction context.
func WithGlobalTx(ctx context.Context, gc *GtxConfig, businessFunc CallbackWithCtx) error {
	return tm.WithGlobalTx(ctx, gc, businessFunc)
}
