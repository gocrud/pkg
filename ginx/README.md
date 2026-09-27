# ginx

Gin 统一响应与错误处理中间件。导入路径为 `github.com/gocrud/pkg/ginx`，包名是 **`ginx`**（与目录一致，无需别名）。

## 最小服务

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/gocrud/pkg/errorx"
    "github.com/gocrud/pkg/ginx"
    "github.com/gocrud/pkg/logx"
)

func main() {
    logger := logx.NewInstance(&logx.Config{
        Level: "info", Target: "console", Format: "text",
    })
    router := gin.New()
    router.Use(gin.Recovery(), ginx.AutoErrorInterceptor(logger))

    router.GET("/health", func(ctx *gin.Context) {
        ginx.Ok(ctx, gin.H{"status": "ok"})
    })
    router.GET("/users/:id", func(ctx *gin.Context) {
        ginx.Fail(ctx, errorx.Define("USER_NOT_FOUND", "用户不存在"))
        return
    })

    if err := router.Run(":8080"); err != nil {
        logger.Fatal().Err(err).Msg("http server stopped")
    }
}
```

## 响应接口

| API | 行为 |
| --- | --- |
| `Ok(ctx, data)` | HTTP 200，`code=SUCCESS`、`msg=success`，携带 data |
| `Msg(ctx, msg)` | HTTP 200，`code=SUCCESS`，自定义 msg |
| `FailParam(ctx)` | 登记 `ERR_PARAM`，消息为“请求参数格式错误”，并 Abort |
| `Fail(ctx, err)` | err 非 nil 时登记错误并 Abort；nil 时不做任何事 |
| `AutoErrorInterceptor(logger)` | 下游执行后，将最后一条登记错误转换为响应 |

`Fail` 和 `FailParam` 本身不写 JSON，必须搭配中间件。`Abort()` 不会退出当前 Go 函数，因此调用后通常需要 `return`。使用 `ShouldBindJSON` 等绑定方法时，应自行检查返回错误，再调用 `FailParam` 或 `Fail`。

统一响应结构为 `Result`：`code`、`msg`，以及带 `omitempty` 的 `data`、`errors`。参数校验响应示例：

```json
{
  "code": "ERR_PARAM",
  "msg": "参数校验未通过",
  "errors": [{"field": "email", "message": "邮箱格式不正确"}]
}
```

## 错误映射与边界

| 错误类型 | HTTP 状态 | 响应 |
| --- | --- | --- |
| 错误链包含 `*veri.ValidationErrors` | 200 | `ERR_PARAM`，保留字段与消息列表 |
| 错误链包含 errorx 业务错误（非 `ERR_SYS`） | 200 | 业务 code 与用户消息 |
| 错误链包含 errorx 业务错误且 code 为 `ERR_SYS` | 500 | `ERR_SYS`，固定消息“系统繁忙，请稍后再试” |
| 其他错误 | 500 | `ERR_SYS`，固定消息“服务异常” |

- 业务错误通过 `errorx.ErrorOf` 沿 `Unwrap` 链查找，`fmt.Errorf("...: %w", bizErr)` 包装后仍能识别。
- 只处理 `ctx.Errors` 的最后一条错误；若响应已写入，则跳过处理。
- 中间件不主动执行参数校验，也不负责 panic 恢复。`gin.Recovery()` 的响应不保证符合 `Result` 结构。
