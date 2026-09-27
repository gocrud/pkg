package errorx_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gocrud/pkg/errorx"
)

var (
	ErrUserNotFound = errorx.Define("USER_NOT_FOUND", "user {{.id}} not found")
	ErrOrderFail    = errorx.Define(10001, "下单失败")
	ErrStockLow     = errorx.Define(int64(20001), "库存不足，剩余 {{.rest}}")
)

func TestDefineCodeField(t *testing.T) {
	if ErrUserNotFound.Code != "USER_NOT_FOUND" {
		t.Fatalf("ErrUserNotFound.Code = %v", ErrUserNotFound.Code)
	}
	if ErrOrderFail.Code != 10001 {
		t.Fatalf("ErrOrderFail.Code = %v", ErrOrderFail.Code)
	}
	if ErrStockLow.Code != int64(20001) {
		t.Fatalf("ErrStockLow.Code = %v", ErrStockLow.Code)
	}
}

func TestErrorSingleLine(t *testing.T) {
	if got := ErrOrderFail.Error(); got != "[10001] 下单失败" {
		t.Fatalf("Error() = %q", got)
	}
	if got := ErrStockLow.Int("rest", 3).Error(); got != "[20001] 库存不足，剩余 3" {
		t.Fatalf("Error() = %q", got)
	}
	// 无参数时占位符为空并折叠空白
	if got := ErrUserNotFound.Error(); got != "[USER_NOT_FOUND] user not found" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestChainedSetters(t *testing.T) {
	err := ErrUserNotFound.Str("id", "u1").Int("age", 18)
	var b *errorx.Builder
	if !errors.As(err, &b) {
		t.Fatalf("expected *errorx.Builder, got %T", err)
	}
	if got := err.Error(); got != "[USER_NOT_FOUND] user u1 not found" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestWrapCauseChain(t *testing.T) {
	dbErr := errors.New("dial failed")
	err := ErrUserNotFound.Str("id", "u1").Wrap(dbErr)
	if !errors.Is(err, dbErr) {
		t.Fatal("errors.Is(err, cause) = false")
	}
	var e *errorx.Error
	if !errors.As(err, &e) {
		t.Fatalf("expected *errorx.Error, got %T", err)
	}
	if e.Unwrap() != dbErr {
		t.Fatal("Unwrap() mismatch")
	}
	if e.CodeStr() != "USER_NOT_FOUND" {
		t.Fatalf("CodeStr() = %q", e.CodeStr())
	}
	if e.Args()["id"] != "u1" {
		t.Fatalf("Args() = %v", e.Args())
	}
}

func TestSentinelIs(t *testing.T) {
	err := ErrUserNotFound.Str("id", "u1")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatal("errors.Is(builder, ErrUserNotFound) = false")
	}
	if !errors.Is(ErrUserNotFound, ErrUserNotFound) {
		t.Fatal("errors.Is(sentinel, sentinel) = false")
	}
	if errors.Is(err, ErrOrderFail) {
		t.Fatal("errors.Is(builder, ErrOrderFail) = true")
	}
}

func TestIsThroughCauseChain(t *testing.T) {
	inner := errorx.Wrap(errors.New("db down"), ErrOrderFail)
	outer := ErrUserNotFound.Str("id", "u1").Wrap(inner)
	if !errors.Is(outer, ErrOrderFail) {
		t.Fatal("errors.Is(outer, innerCode) = false")
	}
	if !errors.Is(outer, ErrUserNotFound) {
		t.Fatal("errors.Is(outer, ownCode) = false")
	}
}

func TestCodeOf(t *testing.T) {
	code, ok := errorx.CodeOf(ErrOrderFail)
	if !ok || code != "10001" {
		t.Fatalf("CodeOf(sentinel) = %q, %v", code, ok)
	}
	code, ok = errorx.CodeOf(ErrUserNotFound.Str("id", "u1"))
	if !ok || code != "USER_NOT_FOUND" {
		t.Fatalf("CodeOf(builder) = %q, %v", code, ok)
	}
	if _, ok := errorx.CodeOf(errors.New("plain")); ok {
		t.Fatal("CodeOf(plain) = true")
	}
}

func TestErrorOfNormalizesAllShapes(t *testing.T) {
	sentinelView, ok := errorx.ErrorOf(ErrOrderFail)
	if !ok || sentinelView.CodeStr() != "10001" {
		t.Fatalf("ErrorOf(sentinel) = %v, %v", sentinelView, ok)
	}
	builderView, ok := errorx.ErrorOf(ErrUserNotFound.Str("id", "u1"))
	if !ok || builderView.Args()["id"] != "u1" {
		t.Fatalf("ErrorOf(builder) = %v, %v", builderView, ok)
	}
	terminal := ErrUserNotFound.Str("id", "u1").Wrap(errors.New("x"))
	terminalView, ok := errorx.ErrorOf(terminal)
	if !ok || terminalView != terminal {
		t.Fatalf("ErrorOf(terminal) = %v, %v", terminalView, ok)
	}
	if _, ok := errorx.ErrorOf(errors.New("plain")); ok {
		t.Fatal("ErrorOf(plain) = true")
	}
}

func TestArgsOverwrite(t *testing.T) {
	err := ErrStockLow.Int("rest", 1).Int("rest", 5)
	view, ok := errorx.ErrorOf(err)
	if !ok || view.Args()["rest"] != 5 {
		t.Fatalf("Args overwrite = %v", view)
	}
	if got := err.Error(); got != "[20001] 库存不足，剩余 5" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestErrorOfWrappedChain(t *testing.T) {
	wrapped := fmt.Errorf("outer: %w", ErrUserNotFound.Str("id", "u1"))
	view, ok := errorx.ErrorOf(wrapped)
	if !ok || view.CodeStr() != "USER_NOT_FOUND" || view.Msg() != "user u1 not found" {
		t.Fatalf("ErrorOf(wrapped) = %v, %v", view, ok)
	}
	if code, ok := errorx.CodeOf(wrapped); !ok || code != "USER_NOT_FOUND" {
		t.Fatalf("CodeOf(wrapped) = %q, %v", code, ok)
	}
}

func TestExtraArgsIgnored(t *testing.T) {
	err := ErrUserNotFound.Str("id", "u9").Any("age", 18)
	if got := err.Error(); got != "[USER_NOT_FOUND] user u9 not found" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestWrapPackageFunc(t *testing.T) {
	if errorx.Wrap(nil, ErrOrderFail) != nil {
		t.Fatal("Wrap(nil) != nil")
	}
	err := errorx.Wrap(errors.New("db down"), ErrOrderFail)
	if got := err.Error(); got != "[10001] 下单失败" {
		t.Fatalf("Error() = %q", got)
	}
	if !errors.Is(err, ErrOrderFail) {
		t.Fatal("errors.Is = false")
	}
}

func TestDefineDynamicCode(t *testing.T) {
	code := errorx.Define("SEATA_BEGIN", "seata begin failed")
	if got := code.Error(); got != "[SEATA_BEGIN] seata begin failed" {
		t.Fatalf("Error() = %q", got)
	}
	num := errorx.Define(500, "内部错误")
	if got := num.Error(); got != "[500] 内部错误" {
		t.Fatalf("Error() = %q", got)
	}
}
