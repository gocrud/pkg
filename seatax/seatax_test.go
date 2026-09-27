package seatax

import (
	"context"
	"database/sql"
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
	seatasql "seata.apache.org/seata-go/v2/pkg/datasource/sql"
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
	be, ok := errorx.ErrorOf(err)
	if !ok {
		t.Fatalf("期望 errorx 业务错误,实际: %v", err)
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
	biz := newBizErr("BIZ_FAIL", "业务失败")
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
		return newBizErr("BIZ_FAIL", "业务失败")
	})
	if bizCode(t, err) != "BIZ_FAIL" {
		t.Fatalf("回滚失败时也应保留业务错误码,实际: %v", err)
	}
}

func TestComposeTxResult(t *testing.T) {
	biz := newBizErr("BIZ_FAIL", "业务失败")
	panicText := "boom" // 非 error panic 值
	phaseErr := errors.New("phase error")

	cases := []struct {
		name     string
		bizErr   error
		panicVal any
		phaseErr error
		wantCode string // 为空表示期望 nil
	}{
		{"提交失败", nil, nil, phaseErr, CodeCommit},
		{"回滚失败保留业务码", biz, nil, phaseErr, "BIZ_FAIL"},
		{"回滚失败且 panic", nil, panicText, phaseErr, CodeInternal},
		{"panic 回滚成功", nil, panicText, nil, errorx.ErrInternal},
		{"业务错误透传", biz, nil, nil, "BIZ_FAIL"},
		{"全部成功", nil, nil, nil, ""},
	}
	for _, c := range cases {
		got := composeTxResult(c.bizErr, c.panicVal, c.phaseErr)
		if c.wantCode == "" {
			if got != nil {
				t.Errorf("%s: 期望 nil,实际 %v", c.name, got)
			}
			continue
		}
		if bizCode(t, got) != c.wantCode {
			t.Errorf("%s: 期望 %s,实际 %v", c.name, c.wantCode, got)
		}
	}
}

func TestWithGlobalTxPropagationNotSupported(t *testing.T) {
	globalMock.reset()
	called := false
	err := WithGlobalTx(context.Background(), "tx-unsupported", func(ctx context.Context) error {
		called = true
		return nil
	}, WithPropagation(PropagationNotSupported))
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
	if _, err := OpenDataSource(ModeAT, DBTypeMySQL, ""); bizCode(t, err) != errorx.ErrParam {
		t.Fatalf("空 DSN 应返回参数错误,实际: %v", err)
	}
	if _, err := OpenDataSource(ModeXA, DBTypePostgres, ""); bizCode(t, err) != errorx.ErrParam {
		t.Fatalf("空 DSN 应返回参数错误,实际: %v", err)
	}
}

func TestOpenGormNilConn(t *testing.T) {
	if _, err := WrapGorm(DBTypeMySQL, nil); bizCode(t, err) != errorx.ErrParam {
		t.Fatalf("nil 连接应返回参数错误,实际: %v", err)
	}
}

func TestOpenGormUnsupportedDBType(t *testing.T) {
	conn, err := sql.Open("mysql", "user:pass@tcp(127.0.0.1:3306)/db")
	if err != nil {
		t.Fatalf("构造测试连接失败: %v", err)
	}
	defer conn.Close()
	if _, err := WrapGorm(DBType("oracle"), conn); err == nil {
		t.Fatal("WrapGorm 不支持的数据库类型应返回错误")
	}
}

func TestDBSpec(t *testing.T) {
	cases := []struct {
		mode   Mode
		dbType DBType
		want   string
	}{
		{ModeAT, DBTypeMySQL, seatasql.SeataATMySQLDriver},
		{ModeXA, DBTypeMySQL, seatasql.SeataXAMySQLDriver},
		{ModeAT, DBTypePostgres, seatasql.SeataATPostgresDriver},
		{ModeXA, DBTypePostgres, seatasql.SeataXAPostgresDriver},
	}
	for _, c := range cases {
		spec, err := dbSpecFor(c.dbType)
		if err != nil {
			t.Errorf("dbSpecFor(%q) 意外错误: %v", c.dbType, err)
			continue
		}
		if got := spec.driver(c.mode); got != c.want {
			t.Errorf("dbSpec(%q).driver(%v) = %q, 期望 %q", c.dbType, c.mode, got, c.want)
		}
	}
	if _, err := dbSpecFor(DBType("oracle")); err == nil {
		t.Fatal("不支持的数据库类型应返回错误")
	}
	for _, s := range []string{"MySQL", "PostgreSQL", "pgsql"} {
		if _, err := dbSpecFor(DBType(s)); err == nil {
			t.Errorf("大小写变体/别名 %q 应返回错误(严格匹配)", s)
		}
	}
	if _, err := OpenGorm(ModeAT, DBType("oracle"), "dsn"); err == nil {
		t.Fatal("OpenGorm 不支持的数据库类型应返回错误")
	}
}

func TestDatabaseOptions(t *testing.T) {
	o := &databaseOptions{mode: ModeAT, dbType: DBTypeMySQL}
	WithMode(ModeXA)(o)
	WithDBType(DBTypePostgres)(o)
	if o.mode != ModeXA || o.dbType != DBTypePostgres {
		t.Errorf("数据库选项未生效: mode=%v dbType=%v", o.mode, o.dbType)
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
	// c.Error + Abort 后由 ginx.AutoErrorInterceptor 统一渲染响应,这里只校验已中止。
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

func TestWithXIDMetadata(t *testing.T) {
	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, "xid-micro")
	out := WithXIDMetadata(ctx)
	md, ok := metadata.FromContext(out)
	if !ok {
		t.Fatal("应写入 outgoing metadata")
	}
	if md[constant.XidKey] != "xid-micro" {
		t.Fatalf("metadata 应携带 XID,实际: %q", md[constant.XidKey])
	}
	// 无 XID 时原样返回
	if out2 := WithXIDMetadata(context.Background()); out2 != context.Background() {
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
