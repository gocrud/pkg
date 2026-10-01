# microx

面向 go-micro v6 的 gRPC server/client 场景(原生 gRPC 兼容)，提供与 `grpcx` 对齐的请求校验与错误转换。go-micro 原生结构化错误 `*errors.Error` 本身就是 JSON 自描述的完整载体：业务码放入 `Reason`，`Code` 字段采用 HTTP 风格并由 `server/grpc` 自动映射为 gRPC status code，因此无需 grpcx 的 trailer 机制。

## 服务端接入

```go
package example

import (
    "github.com/gocrud/pkg/microx"
    "go-micro.dev/v6"
    "go-micro.dev/v6/server"
    grpcServer "go-micro.dev/v6/server/grpc"
)

func NewService() micro.Service {
    service := micro.NewService("helloworld",
        micro.Server(grpcServer.NewServer(
            server.Name("helloworld"),
            server.Address(":8080"),
            grpcServer.Reflection(),
            server.WrapHandler(
                microx.ValidationHandlerWrapper("Health.Check"),
                microx.ErrorHandlerWrapper(),
            ),
        )),
    )
    return service
}
```

`ErrorHandlerWrapper` 统一把 handler 返回的错误转换为 go-micro 结构化错误；也可以不使用它，改为在每个 handler 方法中显式调用 `ToMicroError`。两种方式选一种，避免重复转换。`ValidationHandlerWrapper(skipEndpoints...)` 的跳过列表精确匹配 `req.Endpoint()`，格式如 `Say.Hello`。

## 协议映射

| 服务端错误 | HTTP Code | gRPC code | Reason |
| --- | --- | --- | --- |
| `nil` | 无错误 | 无错误 | 无 |
| `*veri.ValidationErrors` | 400 | `InvalidArgument` | `ERR_PARAM` |
| errorx 业务错误，code 为 `ERR_UNAUTH` | 401 | `Unauthenticated` | `ERR_UNAUTH` |
| errorx 业务错误，其余业务码 | 400 | `InvalidArgument` | 业务码 |
| 其他错误，包括 `ERR_SYS` | 500 | `Internal` | `ERR_SYS` |

字段详情采用 `field:message; field:message` 拼入 `Detail`。内部错误不向客户端暴露原因，统一返回“系统繁忙，请稍后再试”。

协议层错误码定义在本包：`microx.ErrParam`（`ERR_PARAM`）、`microx.ErrUnauthorized`（`ERR_UNAUTH`）、`microx.ErrInternal`（`ERR_SYS`），写入结构化错误的 `Reason`；`errorx` 只提供 `Define` 与消息渲染。

## 客户端还原

`FromMicroError(err)` 返回 `(converted bool, err error)`：用 `errors.As` 提取 `*errors.Error`，`Reason` 非空时还原为 `errorx.Define(reason, detail)`；`Reason` 为空时按 `Code` 兜底(400→`ERR_PARAM`、401→`ERR_UNAUTH`、500→`ERR_SYS`)，其余原样返回，nil 返回 `(false, nil)`。
