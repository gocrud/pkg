package logx

import (
	"github.com/gocrud/kernel"
	"github.com/rs/zerolog"
)

// AddLog 返回一个 kernel 扩展,把 cfg 构造的 zerolog.Logger 注册为单例;
// 调用方已注册 zerolog.Logger 时跳过本次注册(TryProvide 语义)。
// cfg 为 nil 时立即 panic,属注册期错误,尽早暴露:
//
//	kernel.New().
//	    Extend(logx.AddLog(&logx.Config{Level: "info", Target: "both", Format: "text"}))
func AddLog(cfg *Config) kernel.Extension {
	if cfg == nil {
		panic("logx: AddLog 的 cfg 不能为空")
	}
	return func(b *kernel.AppBuilder) {
		b.TryProvide[zerolog.Logger](func() zerolog.Logger {
			return NewInstance(cfg)
		})
	}
}
