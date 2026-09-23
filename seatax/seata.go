// Package seatax 封装 seata.apache.org/seata-go/v2,同时提供 TM(全局事务)与
// RM(AT/XA 数据源、TCC)能力,并针对 gRPC、HTTP(gin)、go-micro 三种协议提供
// XID 传播组件。所有 seata 内部错误统一转换为 errorx.BizError(SEATA_* 错误码),
// 业务错误原样透传,与 grpcx / httpx(ginx) / microx 的统一错误处理链路兼容。
package seatax

import (
	"os"
	"sync/atomic"

	"github.com/gocrud/pkg/errorx"
	"seata.apache.org/seata-go/v2/pkg/client"
)

var initialized atomic.Bool

// Init 使用指定配置文件初始化 seata 客户端。
// 支持 yaml / yml / json / toml 格式,配置结构见 seata 官方 seatago.yml。
// path 为空时回退读取环境变量 SEATA_GO_CONFIG_PATH;两者均缺失或配置非法时
// 返回 CodeConfig 业务错误。SDK 内部各子系统按 sync.Once 幂等初始化,
// 重复调用 Init 不会产生副作用。
func Init(path string) (err error) {
	if initialized.Load() {
		return nil
	}
	defer func() {
		if r := recover(); r != nil {
			err = errorx.E(CodeConfig, "初始化 Seata 客户端失败", toError(r))
		}
	}()
	client.InitPath(path)
	initialized.Store(true)
	return nil
}

// InitFromConf 使用内存中的配置字节初始化 seata 客户端,便于将 seatago.yml
// 内容内嵌进业务应用。配置会先写入临时文件再交给 SDK 加载,初始化完成后删除。
func InitFromConf(conf []byte) (err error) {
	if len(conf) == 0 {
		return errorx.E(CodeConfig, "Seata 配置内容为空")
	}
	file, err := os.CreateTemp("", "seatax-conf-*.yaml")
	if err != nil {
		return errorx.E(CodeConfig, "写入 Seata 临时配置失败", err)
	}
	cleanup := func() {
		_ = file.Close()
		_ = os.Remove(file.Name())
	}
	defer cleanup()
	if _, err = file.Write(conf); err != nil {
		return errorx.E(CodeConfig, "写入 Seata 临时配置失败", err)
	}
	if err = file.Close(); err != nil {
		return errorx.E(CodeConfig, "写入 Seata 临时配置失败", err)
	}
	return Init(file.Name())
}
