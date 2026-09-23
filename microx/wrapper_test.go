package microx

import (
	"context"
	"errors"
	"testing"

	"github.com/gocrud/pkg/errorx"
	"github.com/gocrud/veri"
	"go-micro.dev/v6/codec"
	microerrors "go-micro.dev/v6/errors"
	"go-micro.dev/v6/server"
)

type mockRequest struct {
	endpoint string
	body     interface{}
}

func (r *mockRequest) Service() string           { return "test" }
func (r *mockRequest) Method() string            { return "Test.Call" }
func (r *mockRequest) Endpoint() string          { return r.endpoint }
func (r *mockRequest) ContentType() string       { return "" }
func (r *mockRequest) Header() map[string]string { return nil }
func (r *mockRequest) Body() interface{}         { return r.body }
func (r *mockRequest) Read() ([]byte, error)     { return nil, nil }
func (r *mockRequest) Codec() codec.Reader       { return nil }
func (r *mockRequest) Stream() bool              { return false }

type validatable struct{ err error }

func (v *validatable) Validate() error { return v.err }

func TestValidationHandlerWrapper_PassThrough(t *testing.T) {
	called := false
	next := func(ctx context.Context, req server.Request, rsp interface{}) error {
		called = true
		return nil
	}
	handler := ValidationHandlerWrapper()(next)
	if err := handler(context.Background(), &mockRequest{endpoint: "Test.Call", body: &struct{}{}}, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("next not called")
	}
}

func TestValidationHandlerWrapper_Valid(t *testing.T) {
	called := false
	next := func(ctx context.Context, req server.Request, rsp interface{}) error {
		called = true
		return nil
	}
	handler := ValidationHandlerWrapper()(next)
	if err := handler(context.Background(), &mockRequest{endpoint: "Test.Call", body: &validatable{}}, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("next not called")
	}
}

func TestValidationHandlerWrapper_Invalid(t *testing.T) {
	called := false
	next := func(ctx context.Context, req server.Request, rsp interface{}) error {
		called = true
		return nil
	}
	handler := ValidationHandlerWrapper()(next)
	err := handler(context.Background(), &mockRequest{endpoint: "Test.Call", body: &validatable{err: errorx.E("USER_NOT_FOUND", "用户不存在")}}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	merr, ok := microerrors.As(err)
	if !ok || merr.Reason != "USER_NOT_FOUND" {
		t.Fatalf("got %v", err)
	}
	if called {
		t.Fatal("next should not be called")
	}
}

func TestValidationHandlerWrapper_Skip(t *testing.T) {
	called := false
	next := func(ctx context.Context, req server.Request, rsp interface{}) error {
		called = true
		return nil
	}
	handler := ValidationHandlerWrapper("Test.Call")(next)
	err := handler(context.Background(), &mockRequest{endpoint: "Test.Call", body: &validatable{err: errorx.E("USER_NOT_FOUND", "用户不存在")}}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("next should be called for skipped endpoint")
	}
}

func TestErrorHandlerWrapper_Nil(t *testing.T) {
	next := func(ctx context.Context, req server.Request, rsp interface{}) error { return nil }
	handler := ErrorHandlerWrapper()(next)
	if err := handler(context.Background(), &mockRequest{endpoint: "Test.Call"}, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestErrorHandlerWrapper_BizError(t *testing.T) {
	next := func(ctx context.Context, req server.Request, rsp interface{}) error {
		return errorx.E("USER_NOT_FOUND", "用户不存在")
	}
	handler := ErrorHandlerWrapper()(next)
	err := handler(context.Background(), &mockRequest{endpoint: "Test.Call"}, nil)
	merr := asMicroError(t, err)
	if merr.Reason != "USER_NOT_FOUND" {
		t.Fatalf("reason = %q, want USER_NOT_FOUND", merr.Reason)
	}
}

func TestErrorHandlerWrapper_ValidationErrors(t *testing.T) {
	next := func(ctx context.Context, req server.Request, rsp interface{}) error {
		return &veri.ValidationErrors{Errors: []*veri.FieldError{{Field: "name", Message: "必填"}}}
	}
	handler := ErrorHandlerWrapper()(next)
	err := handler(context.Background(), &mockRequest{endpoint: "Test.Call"}, nil)
	merr := asMicroError(t, err)
	if merr.Reason != errorx.ErrParam || merr.Code != HTTPBadRequest {
		t.Fatalf("got code=%d reason=%q", merr.Code, merr.Reason)
	}
}

func TestErrorHandlerWrapper_Unknown(t *testing.T) {
	next := func(ctx context.Context, req server.Request, rsp interface{}) error {
		return errors.New("boom")
	}
	handler := ErrorHandlerWrapper()(next)
	err := handler(context.Background(), &mockRequest{endpoint: "Test.Call"}, nil)
	merr := asMicroError(t, err)
	if merr.Reason != errorx.ErrInternal || merr.Code != HTTPInternalError {
		t.Fatalf("got code=%d reason=%q", merr.Code, merr.Reason)
	}
}
