package seatax

import (
	"context"
	"fmt"
	"time"

	"github.com/gocrud/pkg/errorx"
	seataTM "seata.apache.org/seata-go/v2/pkg/tm"
)

// Propagation 全局事务传播行为,镜像 seata-go SDK 的传播语义(数值一致)。
type Propagation int8

const (
	// PropagationRequired 默认传播:存在则加入,否则新建。
	PropagationRequired Propagation = 0
	// PropagationRequiresNew 挂起当前事务,新建事务。
	PropagationRequiresNew Propagation = 1
	// PropagationNotSupported 挂起当前事务,以无事务方式执行。
	PropagationNotSupported Propagation = 2
	// PropagationSupports 存在则加入,否则以无事务方式执行。
	PropagationSupports Propagation = 3
	// PropagationNever 存在事务则报错,否则以无事务方式执行。
	PropagationNever Propagation = 4
	// PropagationMandatory 必须存在事务,否则报错。
	PropagationMandatory Propagation = 5
)

// String 返回传播行为的字符串表示。
func (p Propagation) String() string {
	switch p {
	case PropagationRequired:
		return "Required"
	case PropagationRequiresNew:
		return "RequiresNew"
	case PropagationNotSupported:
		return "NotSupported"
	case PropagationSupports:
		return "Supports"
	case PropagationNever:
		return "Never"
	case PropagationMandatory:
		return "Mandatory"
	default:
		return "Unknown"
	}
}

// TxOption 全局事务可选项。
type TxOption func(*seataTM.GtxConfig)

// WithTimeout 设置全局事务超时时间;为 0 时使用 SDK 默认超时。
func WithTimeout(timeout time.Duration) TxOption {
	return func(gc *seataTM.GtxConfig) {
		gc.Timeout = timeout
	}
}

// WithPropagation 设置事务传播行为,默认 PropagationRequired。
func WithPropagation(propagation Propagation) TxOption {
	return func(gc *seataTM.GtxConfig) {
		gc.Propagation = seataTM.Propagation(propagation)
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
func WithGlobalTx(ctx context.Context, name string, fn TxFn, opts ...TxOption) error {
	if fn == nil {
		return newBizErr(errorx.ErrParam, "全局事务业务函数不能为空")
	}
	if name == "" {
		return newBizErr(errorx.ErrParam, "全局事务名称不能为空")
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
	if err := seataTM.Begin(ctx, cfg); err != nil {
		return newBizErr(CodeBegin, "开启全局事务失败", err)
	}
	bizErr, panicVal := runTx(ctx, fn)
	if !seataTM.IsGlobalTx(ctx) {
		if panicVal != nil {
			return newBizErr(errorx.ErrInternal, "业务执行异常", toError(panicVal))
		}
		return bizErr
	}
	phaseErr := seataTM.CommitOrRollback(ctx, panicVal == nil && bizErr == nil)
	return composeTxResult(bizErr, panicVal, phaseErr)
}

// runTx 执行业务回调并收敛 panic:返回业务错误与 panic 值(二者互斥)。
func runTx(ctx context.Context, fn TxFn) (bizErr error, panicVal any) {
	defer func() {
		panicVal = recover()
	}()
	bizErr = fn(ctx)
	return
}

// composeTxResult 根据业务结果与二阶段结果组合最终错误。
func composeTxResult(bizErr error, panicVal any, phaseErr error) error {
	if phaseErr != nil {
		if bizErr == nil && panicVal == nil {
			return newBizErr(CodeCommit, "提交全局事务失败", phaseErr)
		}
		// 业务失败触发回滚、且回滚本身失败:保留业务错误码,回滚失败挂 cause。
		cause := fmt.Errorf("全局事务二阶段处理失败: %v", phaseErr)
		return keepBizError(firstErr(panicVal, bizErr), cause)
	}
	if bizErr == nil && panicVal != nil {
		return newBizErr(errorx.ErrInternal, "业务执行异常", toError(panicVal))
	}
	return bizErr
}
