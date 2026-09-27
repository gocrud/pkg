package errorx_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gocrud/pkg/errorx"
)

func buildStackedErr() error {
	return ErrOrderFail.Stack().Wrap(errors.New("boom"))
}

func TestStackCapture(t *testing.T) {
	err := buildStackedErr()
	var e *errorx.Error
	if !errors.As(err, &e) {
		t.Fatalf("expected *errorx.Error, got %T", err)
	}
	stack := e.StackStr()
	if stack == "" {
		t.Fatal("StackStr() empty")
	}
	if !strings.Contains(stack, "TestStackCapture") {
		t.Fatalf("stack 缺业务帧:\n%s", stack)
	}
	if !strings.Contains(stack, "buildStackedErr") {
		t.Fatalf("stack 缺 helper 帧:\n%s", stack)
	}
	if strings.Contains(stack, "runtime.") {
		t.Fatalf("stack 含 runtime 帧:\n%s", stack)
	}
	if strings.Contains(stack, "errorx.") {
		t.Fatalf("stack 含 errorx 内部帧:\n%s", stack)
	}
	lines := strings.Split(strings.TrimSpace(stack), "\n")
	if len(lines) < 2 {
		t.Fatalf("帧数不足:\n%s", stack)
	}
	if !strings.HasPrefix(lines[0], "├─ ") {
		t.Fatalf("首帧应以 ├─ 开头:\n%s", stack)
	}
	if !strings.Contains(lines[len(lines)-1], "└─ ") {
		t.Fatalf("末帧应以 └─ 标记:\n%s", stack)
	}
}

func TestFormatVerb(t *testing.T) {
	err := ErrUserNotFound.Str("id", "42").Wrap(errors.New("dial failed"))
	single := fmt.Sprintf("%v", err)
	if single != "[USER_NOT_FOUND] user 42 not found" {
		t.Fatalf("%%v = %q", single)
	}
	if got := fmt.Sprintf("%s", err); got != single {
		t.Fatalf("%%s = %q", got)
	}
	full := fmt.Sprintf("%+v", err)
	if full == single {
		t.Fatal("full render should differ from single line")
	}
	if !strings.Contains(full, "caused by: dial failed") {
		t.Fatalf("%%+v 缺 cause 链:\n%s", full)
	}
	// Code 无参数无堆栈时 %+v 退化为单行
	if got := fmt.Sprintf("%+v", ErrOrderFail); got != "[10001] 下单失败" {
		t.Fatalf("Code %%+v = %q", got)
	}
}

func TestFormatFullStack(t *testing.T) {
	err := buildStackedErr()
	full := fmt.Sprintf("%+v", err)
	if !strings.Contains(full, "├─ ") {
		t.Fatalf("%%+v 缺箭头树:\n%s", full)
	}
	if !strings.Contains(full, "caused by: boom") {
		t.Fatalf("%%+v 缺 cause:\n%s", full)
	}
}

func TestFormatNestedCause(t *testing.T) {
	inner := errorx.Wrap(errors.New("db down"), ErrOrderFail)
	outer := ErrUserNotFound.Str("id", "1").Wrap(inner)
	full := fmt.Sprintf("%+v", outer)
	if !strings.Contains(full, "caused by: [10001] 下单失败") {
		t.Fatalf("缺嵌套 cause 消息:\n%s", full)
	}
	if !strings.Contains(full, "└─ caused by: db down") {
		t.Fatalf("缺递归 cause 链:\n%s", full)
	}
}
