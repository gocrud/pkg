package seatax

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocrud/pkg/errorx"
	"go-micro.dev/v6/metadata"
	"go-micro.dev/v6/server"
	"google.golang.org/grpc"
	grpcMetadata "google.golang.org/grpc/metadata"
	"seata.apache.org/seata-go/v2/pkg/constant"
	"seata.apache.org/seata-go/v2/pkg/tm"
)

// mockTM 模拟全局事务管理器。SDK 通过 sync.Once 注册全局单例,因此在
// TestMain 中一次性注入;每个用例重置其状态。
type mockTM struct {
	mu          sync.Mutex
	beginErr    error
	commitErr   error
	rollbackErr error
	committed   bool
	rolledBack  bool
}

func (m *mockTM) Begin(ctx context.Context, timeout time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.beginErr != nil {
		return m.beginErr
	}
	tm.SetXID(ctx, "test-xid")
	return nil
}

func (m *mockTM) Commit(ctx context.Context, gtr *tm.GlobalTransaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.committed = true
	return m.commitErr
}

func (m *mockTM) Rollback(ctx context.Context, gtr *tm.GlobalTransaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rolledBack = true
	return m.rollbackErr
}

func (m *mockTM) GlobalReport(ctx context.Context, gtr *tm.GlobalTransaction) (interface{}, error) {
	return nil, nil
}

func (m *mockTM) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.beginErr, m.commitErr, m.rollbackErr = nil, nil, nil
	m.committed, m.rolledBack = false, false
}

var globalMock = &mockTM{}

func TestMain(m *testing.M) {
	_ = os.Unsetenv("SEATA_GO_CONFIG_PATH")
	tm.SetGlobalTransactionManager(globalMock)
	os.Exit(m.Run())
}

func bizCode(t *testing.T, err error) string {
	t.Helper()
	var be errorx.BizError
	if !errors.As(err, &be) {
		t.Fatalf("期望 BizError,实际: %v", err)
	}
	return be.CodeStr()
}

// ---- TM:WithGlobalTx ----

func TestWithGlobalTxCommit(t *testing.T) {
	globalMock.reset()
	err := WithGlobalTx(context.Background(), "tx-commit", func(ctx context.Context) error {
		if GetXID(ctx) != "test-xid" {
			t.Errorf("业务回调内应能读取 XID,实际: %q", GetXID(ctx))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("预期提交成功,实际错误: %v", err)
	}
	if !globalMock.committed || globalMock.rolledBack {
		t.Fatalf("预期仅提交,committed=%v rolledBack=%v", globalMock.committed, globalMock.rolledBack)
	}
}

func TestWithGlobalTxRollbackOnBizError(t *testing.T) {
	globalMock.reset()
	biz := errorx.E("BIZ_FAIL", "业务失败")
	err := WithGlobalTx(context.Background(), "tx-rollback", func(ctx context.Context) error {
		return biz
	})
	if bizCode(t, err) != "BIZ_FAIL" {
		t.Fatalf("业务错误码应原样透传,实际错误: %v", err)
	}
	if !globalMock.rolledBack || globalMock.committed {
		t.Fatalf("预期仅回滚,committed=%v rolledBack=%v", globalMock.committed, globalMock.rolledBack)
	}
}

func TestWithGlobalTxPanic(t *testing.T) {
	globalMock.reset()
	err := WithGlobalTx(context.Background(), "tx-panic", func(ctx context.Context) error {
		panic("boom")
	})
	if err == nil {
		t.Fatal("业务 panic 应转换为错误")
	}
	if code := bizCode(t, err); code != errorx.ErrInternal {
		t.Fatalf("panic 应转换为内部错误码,实际: %s", code)
	}
	if !globalMock.rolledBack {
		t.Fatal("panic 后应回滚")
	}
}

func TestWithGlobalTxNilFn(t *testing.T) {
	err := WithGlobalTx(context.Background(), "tx-nil", nil)
	if bizCode(t, err) != errorx.ErrParam {
		t.Fatalf("nil 回调应返回参数错误,实际: %v", err)
	}
}

func TestWithGlobalTxEmptyName(t *testing.T) {
	err := WithGlobalTx(context.Background(), "", func(ctx context.Context) error { return nil })
	if bizCode(t, err) != errorx.ErrParam {
		t.Fatalf("空事务名应返回参数错误,实际: %v", err)
	}
}

func TestWithGlobalTxBeginFailure(t *testing.T) {
	globalMock.reset()
	globalMock.beginErr = errors.New("tc unreachable")
	err := WithGlobalTx(context.Background(), "tx-begin", func(ctx context.Context) error { return nil })
	if bizCode(t, err) != CodeBegin {
		t.Fatalf("开启失败应返回 %s,实际: %v", CodeBegin, err)
	}
}

func TestWithGlobalTxCommitFailure(t *testing.T) {
	globalMock.reset()
	globalMock.commitErr = errors.New("commit timeout")
	err := WithGlobalTx(context.Background(), "tx-commit-fail", func(ctx context.Context) error { return nil })
	if bizCode(t, err) != CodeCommit {
		t.Fatalf("提交失败应返回 %s,实际: %v", CodeCommit, err)
	}
}

func TestWithGlobalTxRollbackFailureKeepsBizError(t *testing.T) {
	globalMock.reset()
	globalMock.rollbackErr = errors.New("rollback timeout")
	err := WithGlobalTx(context.Background(), "tx-rb-fail", func(ctx context.Context) error {
		return errorx.E("BIZ_FAIL", "业务失败")
	})
	if bizCode(t, err) != "BIZ_FAIL" {
		t.Fatalf("回滚失败时也应保留业务错误码,实际: %v", err)
	}
}

func TestWithGlobalTxPropagationNotSupported(t *testing.T) {
	globalMock.reset()
	called := false
	err := WithGlobalTx(context.Background(), "tx-unsupported", func(ctx context.Context) error {
		called = true
		return nil
	}, WithPropagation(tm.NotSupported))
	if err != nil {
		t.Fatalf("NotSupported 传播下应直接执行,实际错误: %v", err)
	}
	if !called {
		t.Fatal("NotSupported 传播下业务应被执行")
	}
	if globalMock.committed || globalMock.rolledBack {
		t.Fatal("NotSupported 传播下不应提交或回滚")
	}
}

// ---- Init ----

func TestInitMissingConfig(t *testing.T) {
	_ = os.Unsetenv("SEATA_GO_CONFIG_PATH")
	err := Init("")
	if err == nil {
		t.Fatal("缺少配置应返回错误")
	}
	if bizCode(t, err) != CodeConfig {
		t.Fatalf("应返回 %s,实际: %v", CodeConfig, err)
	}
}

func TestInitFromConfEmpty(t *testing.T) {
	if err := InitFromConf(nil); bizCode(t, err) != CodeConfig {
		t.Fatalf("空配置应返回 %s,实际: %v", CodeConfig, err)
	}
}

// ---- RM 数据源 ----

func TestOpenSeataDBEmptyDSN(t *testing.T) {
	if _, err := OpenATMySQL(""); bizCode(t, err) != errorx.ErrParam {
		t.Fatalf("空 DSN 应返回参数错误,实际: %v", err)
	}
	if _, err := OpenXAPostgres(""); bizCode(t, err) != errorx.ErrParam {
		t.Fatalf("空 DSN 应返回参数错误,实际: %v", err)
	}
}

func TestDriverFor(t *testing.T) {
	cases := []struct {
		mode    Mode
		dialect string
		want    string
	}{
		{ModeAT, "mysql", DriverATMySQL},
		{ModeXA, "MySQL", DriverXAMySQL},
		{ModeAT, "postgres", DriverATPostgres},
		{ModeXA, "PostgreSQL", DriverXAPostgres},
		{ModeAT, "pgsql", DriverATPostgres},
	}
	for _, c := range cases {
		got, err := driverFor(c.mode, c.dialect)
		if err != nil || got != c.want {
			t.Errorf("driverFor(%v,%q) = %q,%v, 期望 %q", c.mode, c.dialect, got, err, c.want)
		}
	}
	if _, err := driverFor(ModeAT, "oracle"); err == nil {
		t.Fatal("不支持的方言应返回错误")
	}
	if _, err := OpenGorm(ModeAT, "oracle", "dsn"); err == nil {
		t.Fatal("OpenGorm 不支持的方言应返回错误")
	}
}

func TestParseSeataDriver(t *testing.T) {
	if mode, dialect, err := parseSeataDriver(DriverXAMySQL); err != nil || mode != ModeXA || dialect != "mysql" {
		t.Errorf("parseSeataDriver(%s) 解析错误: %v %v %q", DriverXAMySQL, mode, err, dialect)
	}
	if _, _, err := parseSeataDriver("no-such-driver"); err == nil {
		t.Fatal("未知驱动名应返回错误")
	}
}

// ---- TCC ----

func TestNewTCCProxyInvalidService(t *testing.T) {
	if _, err := NewTCCProxy(struct{}{}); err == nil {
		t.Fatal("非法 TCC 服务应返回错误")
	} else if bizCode(t, err) != CodeRegister {
		t.Fatalf("注册失败应返回 %s,实际: %v", CodeRegister, err)
	}
}

// ---- gin 中间件 ----

func TestGinMiddlewareAllowMissingXID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handlerRan := false
	r.GET("/ping", GinTransactionMiddleware(WithAllowMissingXID()), func(c *gin.Context) {
		handlerRan = true
		if GetXID(c.Request.Context()) != "" {
			t.Errorf("无 XID 时上下文不应有 XID,实际: %q", GetXID(c.Request.Context()))
		}
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("宽松模式应放行,HTTP 状态: %d", w.Code)
	}
	if !handlerRan {
		t.Fatal("宽松模式下 handler 应执行")
	}
}

func TestGinMiddlewareRestoreXID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/ping", GinTransactionMiddleware(), func(c *gin.Context) {
		if GetXID(c.Request.Context()) != "xid-123" {
			t.Errorf("应恢复 XID,实际: %q", GetXID(c.Request.Context()))
		}
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set(constant.XidKey, "xid-123")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("携带 XID 应放行,HTTP 状态: %d", w.Code)
	}
}

func TestGinMiddlewareStrictMissingXID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handlerRan := false
	r.GET("/ping", GinTransactionMiddleware(), func(c *gin.Context) {
		handlerRan = true
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	r.ServeHTTP(w, req)
	if handlerRan {
		t.Fatal("默认严格模式缺 XID 时 handler 不应执行")
	}
	// c.Error + Abort 后由 httpx.AutoErrorInterceptor 统一渲染响应,这里只校验已中止。
	if w.Code == http.StatusOK && w.Body.Len() == 0 {
		t.Log("默认严格模式已中止,响应渲染由业务层错误中间件完成")
	}
}

func TestGinMiddlewareStrictXIDPresent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/ping", GinTransactionMiddleware(), func(c *gin.Context) {
		if GetXID(c.Request.Context()) != "xid-456" {
			t.Errorf("默认严格模式携带 XID 时应恢复,实际: %q", GetXID(c.Request.Context()))
		}
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set(constant.XidKeyLowercase, "xid-456")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("默认严格模式携带 XID 应放行,HTTP 状态: %d", w.Code)
	}
}

// ---- gRPC 拦截器 ----

func TestGRPCServerInterceptorRestoresXID(t *testing.T) {
	md := grpcMetadata.Pairs(constant.XidKey, "xid-grpc")
	ctx := grpcMetadata.NewIncomingContext(context.Background(), md)
	interceptor := ServerTransactionInterceptor()
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req interface{}) (interface{}, error) {
		if GetXID(ctx) != "xid-grpc" {
			t.Errorf("服务端应恢复 XID,实际: %q", GetXID(ctx))
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("拦截器执行失败: %v", err)
	}
}

func TestGRPCClientInterceptorInjectsXID(t *testing.T) {
	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, "xid-grpc")
	interceptor := ClientTransactionInterceptor()
	err := interceptor(ctx, "/svc/Method", nil, nil, nil,
		func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			md, ok := grpcMetadata.FromOutgoingContext(ctx)
			if !ok {
				t.Fatal("客户端应注入 outgoing metadata")
			}
			if got := md.Get(constant.XidKey); len(got) == 0 || got[0] != "xid-grpc" {
				t.Fatalf("outgoing metadata 应携带 XID,实际: %v", got)
			}
			return nil
		})
	if err != nil {
		t.Fatalf("拦截器执行失败: %v", err)
	}
}

// ---- go-micro 包装器 ----

func TestMicroServerWrapperRestoresXID(t *testing.T) {
	ctx := metadata.Set(context.Background(), constant.XidKey, "xid-micro")
	wrapper := MicroServerTransactionWrapper()
	handler := wrapper(func(ctx context.Context, req server.Request, rsp interface{}) error {
		if GetXID(ctx) != "xid-micro" {
			t.Errorf("服务端应恢复 XID,实际: %q", GetXID(ctx))
		}
		return nil
	})
	if err := handler(ctx, nil, nil); err != nil {
		t.Fatalf("handler 执行失败: %v", err)
	}
}

func TestMicroServerWrapperNoXID(t *testing.T) {
	wrapper := MicroServerTransactionWrapper()
	handler := wrapper(func(ctx context.Context, req server.Request, rsp interface{}) error {
		if GetXID(ctx) != "" {
			t.Errorf("无 XID 时不应恢复,实际: %q", GetXID(ctx))
		}
		return nil
	})
	if err := handler(context.Background(), nil, nil); err != nil {
		t.Fatalf("handler 执行失败: %v", err)
	}
}

func TestWithXIDContext(t *testing.T) {
	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, "xid-micro")
	out := WithXIDContext(ctx)
	md, ok := metadata.FromContext(out)
	if !ok {
		t.Fatal("应写入 outgoing metadata")
	}
	if md[constant.XidKey] != "xid-micro" {
		t.Fatalf("metadata 应携带 XID,实际: %q", md[constant.XidKey])
	}
	// 无 XID 时原样返回
	if out2 := WithXIDContext(context.Background()); out2 != context.Background() {
		t.Error("无 XID 时应原样返回 context")
	}
}

// ---- XID 辅助函数 ----

func TestXIDHelpers(t *testing.T) {
	ctx := InitSeataContext(context.Background())
	if !IsSeataContext(ctx) {
		t.Fatal("InitSeataContext 后应可识别 seata 上下文")
	}
	if IsGlobalTx(ctx) {
		t.Fatal("未绑定 XID 时不应处于全局事务")
	}
	SetXID(ctx, "xid-helper")
	if GetXID(ctx) != "xid-helper" {
		t.Fatalf("SetXID/GetXID 不匹配,实际: %q", GetXID(ctx))
	}
	if !IsGlobalTx(ctx) {
		t.Fatal("绑定 XID 后应处于全局事务")
	}
}
