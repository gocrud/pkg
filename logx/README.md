# logx

基于 zerolog 的结构化日志初始化与输出。

`NewInstance(cfg *Config)` 返回 **`zerolog.Logger` 值类型**。cfg 必须非 nil；配置加载由应用负责。

| Config 字段 | 配置键 | 行为 |
| --- | --- | --- |
| `Level` | `level` | zerolog 等级，例如 `debug`、`info`、`warn`、`error`；解析失败回退到 info |
| `Target` | `target` | `console`、`file`、`both`；其他值回退到标准输出 |
| `Format` | `format` | 仅在 console 分支中，`text` 启用可读格式；其他值为 JSON |
| `FilePath` | `file_path` | 文件路径，交给 lumberjack 处理 |
| `MaxBackups` | `max_backups` | 保留旧文件数量，交给 lumberjack 处理 |
| `MaxSize` | `max_size` | 单文件最大大小，单位 MB，交给 lumberjack 处理 |

字段同时提供 `mapstructure` 和 `json` 标签。文件输出始终为 JSON，并启用 `LocalTime` 与 `Compress`；例如 `Target=both, Format=text` 表示控制台可读文本、文件 JSON。Target / Format 按精确字符串匹配，不自动规范化。

配置示例，需由消费方解析到 `logx.Config`：

```json
{
  "level": "info",
  "target": "both",
  "format": "text",
  "file_path": "logs/app.log",
  "max_backups": 7,
  "max_size": 100
}
```

日志自动包含 timestamp 和 caller。`NewInstance` 会将全局 `zerolog.TimeFieldFormat` 设置为 `time.RFC3339Nano`，建议在启动阶段完成初始化。返回值不暴露统一的 Close 方法。

## IoC 注册

| 注册扩展 | 服务类型 | 构造依赖 |
| --- | --- | --- |
| `logx.AddLog(cfg)` | `zerolog.Logger` | 非 nil 的 cfg |

`AddLog` 返回 `ioc.ServiceCollectionExtension`，使用 `TryAddSingleton` 注册。
