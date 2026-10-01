// Package logx 基于 zerolog 提供结构化日志初始化。
//
// NewInstance(cfg *Config) 返回 zerolog.Logger 值类型,cfg 必须非 nil:
//
//	logger := logx.NewInstance(&logx.Config{Level: "info", Target: "both", Format: "text"})
//	logger.Info().Str("svc", "order").Msg("started")
//
// Config 字段(括号内为配置键):Level(level)、Target(target:console/file/both)、
// Format(format:text/JSON)、FilePath(file_path)、MaxBackups(max_backups)、
// MaxSize(max_size,单位 MB)。文件输出始终为 JSON 并启用 lumberjack 轮转;
// AddLog(cfg) 返回 kernel.Extension,把 zerolog.Logger 注册为单例。详见本包 README。
package logx
