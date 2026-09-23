package seatax

import (
	"context"
	"fmt"
	"time"

	"github.com/gocrud/pkg/errorx"
	seataTM "seata.apache.org/seata-go/v2/pkg/tm"
)

// TxOption 全局事务可选项。
type TxOption func(*seataTM.GtxConfig)

// WithTimeout 设置全局事务超时时间;为 0 时使用 SDK 默认超时。
func WithTimeout(timeout time.Duration) TxOption {
	return func(gc *seataTM.GtxConfig) {
		gc.Timeout = timeout
	}
}

// WithPropagation 设置事务传播行为,默认 seataTM.Required。
func WithPropagation(propagation seataTM.Propagation) TxOption {
	return func(gc *seataTM.GtxConfig) {
		gc.Propagation = propagation
	}
}

// WithLockRetry 设置全局锁重试间隔与次数。
func WithLockRetry(interval time.Duration, times int16) TxOption {
	return func(gc *seataTM.GtxConfig) {
		gc.LockRetryInternal = interval
		gc.LockRetryTimes = times
	}
}

// TxFn 全局事务业务回调。回调内既可以直接操作 seata 数据源,也可以通过
// gRPC / HTTP / go-micro 调用下游服务(seatax 传播组件自动携带 XID)。
type TxFn func(ctx context.Context) error

// WithGlobalTx 开启一个全局事务执行业务回调,按执行结果提交或回滚。
//   - 业务函数返回的错误原样透传(不改变错误码),便于下游统一错误链路渲染;
//   - 开启失败返回 CodeBegin,提交失败返回 CodeCommit,回滚失败返回 CodeRollback;
//   - 业务 panic 会触发回滚,并转换为 errorx 内部错误(Code 为 errorx.ErrInternal)。
func WithGlobalTx(ctx context.Context, name string, fn TxFn, opts ...TxOption) (err error) {
	if fn == nil {
		return errorx.E(errorx.ErrParam, "全局事务业务函数不能为空")
	}
	if name == "" {
		return errorx.E(errorx.ErrParam, "全局事务名称不能为空")
	}
	cfg := &seataTM.GtxConfig{Name: name}
	for _, opt := range opts {
		opt(cfg)
	}
	if !seataTM.IsSeataContext(ctx) {
		ctx = seataTM.InitSeataContext(ctx)
	}
	if seataTM.IsGlobalTx(ctx) {
		seataTM.ClearTxConf(ctx)
	}
	if err = seataTM.Begin(ctx, cfg); err != nil {
		return errorx.E(CodeBegin, "开启全局事务失败", err)
	}
	defer func() {
		deferErr := recover()
		if !seataTM.IsGlobalTx(ctx) {
			if deferErr != nil {
				err = errorx.E(errorx.ErrInternal, "业务执行异常", toError(deferErr))
			}
			return
		}
		if phaseErr := seataTM.CommitOrRollback(ctx, deferErr == nil && err == nil); phaseErr != nil {
			if err == nil && deferErr == nil {
				err = errorx.E(CodeCommit, "提交全局事务失败", phaseErr)
			} else {
				// 业务失败触发回滚、且回滚本身失败:保留业务错误码,回滚失败挂 cause。
				cause := fmt.Errorf("全局事务二阶段处理失败: %v", phaseErr)
				err = keepBizError(firstErr(deferErr, err), cause)
			}
		} else if err == nil && deferErr != nil {
			err = errorx.E(errorx.ErrInternal, "业务执行异常", toError(deferErr))
		}
	}()
	return fn(ctx)
}

// InitSeataContext 初始化 seata 上下文(透传 SDK 实现)。
func InitSeataContext(ctx context.Context) context.Context {
	return seataTM.InitSeataContext(ctx)
}

// IsSeataContext 判断 ctx 是否已初始化 seata 上下文。
func IsSeataContext(ctx context.Context) bool {
	return seataTM.IsSeataContext(ctx)
}

// IsGlobalTx 判断当前 ctx 是否处于全局事务中(已绑定 XID)。
func IsGlobalTx(ctx context.Context) bool {
	return seataTM.IsGlobalTx(ctx)
}

// GetXID 获取当前上下文中的全局事务 XID,不存在时返回空字符串。
func GetXID(ctx context.Context) string {
	return seataTM.GetXID(ctx)
}

// SetXID 向 seata 上下文绑定全局事务 XID。ctx 须先经 InitSeataContext 初始化,
// 否则调用无效果(SDK 行为)。
func SetXID(ctx context.Context, xid string) {
	seataTM.SetXID(ctx, xid)
}
