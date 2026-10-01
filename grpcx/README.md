# grpcx

gRPC Unary 请求校验与 errorx 错误的双向转换。

## 服务端接入

```go
package example

import (
    "context"

    "github.com/gocrud/pkg/grpcx"
    "google.golang.org/grpc"
)

func NewServer() *grpc.Server {
    return grpc.NewServer(grpc.ChainUnaryInterceptor(
        grpcx.UnaryServerValidationInterceptor("/grpc.health.v1.Health/Check"),
        func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
            response, err := handler(ctx, req)
            return response, grpcx.ToGRPCError(ctx, err)
        },
    ))
}
```

上面的第二个拦截器是消费方示例代码，不是本库导出的 API。它统一转换 handler 返回的错误；也可以不添加它，改为在每个 RPC 方法中显式调用 `ToGRPCError`。两种方式选一种，避免重复转换。

`UnaryServerValidationInterceptor(skipMethods...)`：

- 仅用于 Unary RPC，不支持流式 RPC 校验。
- 请求实现 `Validate() error` 时执行校验，否则直接放行。
- 跳过列表精确匹配 `info.FullMethod`，格式为 `/package.Service/Method`。
- 只转换校验阶段返回的错误，**不会转换 handler 返回的错误**。

## 协议映射

| 服务端错误 | gRPC code | trailer |
| --- | --- | --- |
| `nil` | 无错误 | 不设置 |
| 错误链包含 `*veri.ValidationErrors` | `InvalidArgument` | `ERR_PARAM`、固定原因、字段详情 |
| 错误链包含 errorx 业务错误，且 code 非 `ERR_SYS` | `Aborted` | 业务 code 与用户消息 |
| 其他错误，包括 `ERR_SYS` | `Internal` | `ERR_SYS`、固定原因 |

Trailer 键为 `HeaderBizCode` (`x-biz-code`)、`HeaderBizReason` (`x-biz-reason`)、`HeaderBizDetail` (`x-biz-detail`)。字段详情采用 `field:message; field:message` 文本，不是 JSON。

注意：手工创建 `errorx.Define(grpcx.ErrParam, ...)` 属于普通业务错误，在 gRPC 中映射为 `Aborted`；只有 `*veri.ValidationErrors` 分支映射为 `InvalidArgument`。已有 gRPC status 错误若不满足业务错误条件，也会被转换为 `Internal`，不会原样透传。`ToGRPCError` 不记录日志，应由服务端自行记录内部原因。

协议层错误码定义在本包：`grpcx.ErrParam`（`ERR_PARAM`）、`grpcx.ErrInternal`（`ERR_SYS`），作为 `x-biz-code` trailer 的取值；`errorx` 只提供 `Define` 与消息渲染。

## 客户端还原

```go
package example

import (
    "context"

    "github.com/gocrud/pkg/grpcx"
    "google.golang.org/grpc"
    "google.golang.org/grpc/metadata"
)

func Invoke(ctx context.Context, conn grpc.ClientConnInterface, method string, request, response any) error {
    var trailer metadata.MD
    err := conn.Invoke(ctx, method, request, response, grpc.Trailer(&trailer))
    _, translated := grpcx.FromGRPCError(err, trailer)
    return translated
}
```

生成的客户端方法同样可以追加 `grpc.Trailer(&trailer)` 调用选项。`FromGRPCError` 返回 `(converted bool, err error)`：仅对 `Aborted`、`Internal`、`InvalidArgument` 且存在非空 `x-biz-code` 的错误返回 `true` 并构造 `errorx.Define(bizCode, bizReason)`；其他错误原样返回，nil 返回 `(false, nil)`。当前不会还原 `x-biz-detail`，也不会保留原 gRPC 错误作为 cause。
