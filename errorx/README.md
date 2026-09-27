# errorx

声明式、类型安全的业务错误码定义。错误携带「错误码 + 默认模板 + 命名参数 + 底层 cause(可选) + 堆栈(可选)」，渲染时按 `{{.name}}` 命名参数填充模板。

## 定义错误码

```go
var ErrUserNotFound = errorx.Define("USER_NOT_FOUND", "user {{.id}} not found")

var ErrOrderFail = errorx.Define(10001, "下单失败") // 数字码
```

- `Define` 支持字符串与数字码（`CodeValue = ~string | ~int | ~int32 | ~int64 | ~uint | ~uint32 | ~uint64`）。
- `*Code[T]` 本身实现 `error`，无参数错误可直接 `return ErrX`；`ErrX.Code` 保留原类型。
- 协议层通用码：`errorx.ErrOK`（`SUCCESS`）、`errorx.ErrParam`（`ERR_PARAM`）、`errorx.ErrUnauthorized`（`ERR_UNAUTH`）、`errorx.ErrInternal`（`ERR_SYS`），供 `ginx` / `grpcx` / `microx` 等出口层统一归类。

## 链式构建

```go
return ErrUserNotFound.Str("id", id)                 // *Builder，也实现 error
return ErrUserNotFound.Str("id", id).Int("age", 18)  // 链式追加参数
return ErrUserNotFound.Str("id", id).Wrap(err)       // 终结：附加 cause
return ErrUserNotFound.Stack()                       // 显式捕获完整堆栈
return ErrUserNotFound.Wrap(err)                     // 无参直接附加 cause
return ErrUserNotFound.Stack().Str("id", id).Wrap(err)
```

- setter：`Str`/`Int`/`Int64`/`Any`，同名参数后设覆盖先设。
- `Stack()` 按需捕获从应用入口到调用点的完整堆栈（排除 Go 标准库/运行时与 errorx 内部帧）。

## 应用层判定

```go
errors.Is(err, ErrUserNotFound)   // sentinel 匹配（沿 cause 链递归）
code, ok := errorx.CodeOf(err)    // 规范串码（数字 strconv 化）
e, ok := errorx.ErrorOf(err)      // 归一化视图（沿 Unwrap 链查找）：CodeStr/Args/Msg/StackStr
```

## 渲染

| 形式 | 输出 | 用途 |
| --- | --- | --- |
| `Error()` / `%v` | `[10001] 下单失败`（单行） | 协议出口、日志 message |
| `%+v` | 单行消息 + 箭头树堆栈 + `caused by:` 链 | 调试 |
| `StackStr()` | 纯堆栈文本 | 日志单独字段 |

`%+v` 示例：

```text
[10001] 用户 42 不存在
├─ main.main                               cmd/server/start.go:20
│  └─ (*UserService).Get                   internal/service/user.go:88
│     └─ (*UserService).queryByID          internal/service/user.go:123
caused by: dial tcp 127.0.0.1:5432: connect: connection refused
```
