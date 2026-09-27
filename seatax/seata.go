package seatax

import (
	"os"
	"sync/atomic"

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
			err = newBizErr(CodeConfig, "初始化 Seata 客户端失败", toError(r))
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
		return newBizErr(CodeConfig, "Seata 配置内容为空")
	}
	file, err := os.CreateTemp("", "seatax-conf-*.yaml")
	if err != nil {
		return newBizErr(CodeConfig, "写入 Seata 临时配置失败", err)
	}
	cleanup := func() {
		_ = file.Close()
		_ = os.Remove(file.Name())
	}
	defer cleanup()
	if _, err = file.Write(conf); err != nil {
		return newBizErr(CodeConfig, "写入 Seata 临时配置失败", err)
	}
	if err = file.Close(); err != nil {
		return newBizErr(CodeConfig, "写入 Seata 临时配置失败", err)
	}
	return Init(file.Name())
}
