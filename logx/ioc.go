package logx

import (
	"github.com/gocrud/ioc"
	"github.com/rs/zerolog"
)

func AddLog(cfg *Config) ioc.ServiceCollectionExtension {
	return func(sc *ioc.ServiceCollection) *ioc.ServiceCollection {
		sc.TryAddSingleton[zerolog.Logger](func() zerolog.Logger {
			return NewInstance(cfg)
		})
		return sc
	}
}
