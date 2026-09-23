package seatax

import (
	"github.com/gocrud/pkg/errorx"
	seataRM "seata.apache.org/seata-go/v2/pkg/rm"
	seataTCC "seata.apache.org/seata-go/v2/pkg/rm/tcc"
	seataTM "seata.apache.org/seata-go/v2/pkg/tm"
)

// TwoPhaseAction 业务 TCC 服务接口,实现后交给 NewTCCProxy 注册:
//
//	type OrderTCC struct{}
//
//	func (*OrderTCC) GetActionName() string { return "orderTCC" }
//	func (*OrderTCC) Prepare(ctx context.Context, params interface{}) (bool, error) { ... }
//	func (*OrderTCC) Commit(ctx context.Context, bac *seatax.BusinessActionContext) (bool, error) { ... }
//	func (*OrderTCC) Rollback(ctx context.Context, bac *seatax.BusinessActionContext) (bool, error) { ... }
type TwoPhaseAction = seataRM.TwoPhaseInterface

// TCCServiceProxy seata TCC 服务代理。
type TCCServiceProxy = seataTCC.TCCServiceProxy

// BusinessActionContext TCC 二阶段上下文,透传 SDK 类型,供 Commit/Rollback 读取
// Xid、BranchId、ActionContext 等。
type BusinessActionContext = seataTM.BusinessActionContext

// NewTCCProxy 解析并注册 TCC 服务资源,返回代理。
// 业务在一阶段调用 proxy.Prepare(ctx, params),二阶段由 SDK 回调服务的
// Commit / Rollback。注册失败返回 CodeRegister 业务错误。
// 注意:SDK 对非指针入参等非法服务会直接 panic,这里统一收敛为业务错误。
func NewTCCProxy(service interface{}) (proxy *TCCServiceProxy, err error) {
	defer func() {
		if r := recover(); r != nil {
			proxy = nil
			err = errorx.E(CodeRegister, "注册 TCC 资源失败", toError(r))
		}
	}()
	p, e := seataTCC.NewTCCServiceProxy(service)
	if e != nil {
		return nil, errorx.E(CodeRegister, "注册 TCC 资源失败", e)
	}
	return p, nil
}
