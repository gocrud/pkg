package microx

import (
	"errors"
	"strings"
	"testing"

	"github.com/gocrud/pkg/errorx"
	"github.com/gocrud/veri"
	microerrors "go-micro.dev/v6/errors"
)

func asMicroError(t *testing.T, err error) *microerrors.Error {
	t.Helper()
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	merr, ok := microerrors.As(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T: %v", err, err)
	}
	return merr
}

func TestToMicroError_Nil(t *testing.T) {
	if err := ToMicroError(nil); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestToMicroError_ValidationErrors(t *testing.T) {
	err := ToMicroError(&veri.ValidationErrors{Errors: []*veri.FieldError{
		{Field: "name", Message: "用户名不能为空"},
		{Field: "age", Message: "年龄必须大于 0"},
	}})
	merr := asMicroError(t, err)
	if merr.Code != HTTPBadRequest {
		t.Fatalf("code = %d, want %d", merr.Code, HTTPBadRequest)
	}
	if merr.Reason != errorx.ErrParam {
		t.Fatalf("reason = %q, want %q", merr.Reason, errorx.ErrParam)
	}
	if !strings.Contains(merr.Detail, "name:用户名不能为空") || !strings.Contains(merr.Detail, "age:年龄必须大于 0") {
		t.Fatalf("detail = %q, missing field errors", merr.Detail)
	}
}

func TestToMicroError_BizCustomCode(t *testing.T) {
	merr := asMicroError(t, ToMicroError(errorx.E("USER_NOT_FOUND", "用户不存在")))
	if merr.Code != HTTPBadRequest {
		t.Fatalf("code = %d, want %d", merr.Code, HTTPBadRequest)
	}
	if merr.Reason != "USER_NOT_FOUND" {
		t.Fatalf("reason = %q, want USER_NOT_FOUND", merr.Reason)
	}
	if merr.Detail != "用户不存在" {
		t.Fatalf("detail = %q", merr.Detail)
	}
}

func TestToMicroError_BizUnauthorized(t *testing.T) {
	merr := asMicroError(t, ToMicroError(errorx.E(errorx.ErrUnauthorized, "请先登录")))
	if merr.Code != HTTPUnauthorized {
		t.Fatalf("code = %d, want %d", merr.Code, HTTPUnauthorized)
	}
	if merr.Reason != errorx.ErrUnauthorized {
		t.Fatalf("reason = %q, want %q", merr.Reason, errorx.ErrUnauthorized)
	}
}

func TestToMicroError_BizParam(t *testing.T) {
	merr := asMicroError(t, ToMicroError(errorx.E(errorx.ErrParam, "参数错误")))
	if merr.Code != HTTPBadRequest || merr.Reason != errorx.ErrParam {
		t.Fatalf("got code=%d reason=%q", merr.Code, merr.Reason)
	}
}

func TestToMicroError_InternalMasked(t *testing.T) {
	merr := asMicroError(t, ToMicroError(errorx.E(errorx.ErrInternal, "数据库连接失败")))
	if merr.Code != HTTPInternalError || merr.Reason != errorx.ErrInternal {
		t.Fatalf("got code=%d reason=%q", merr.Code, merr.Reason)
	}
	if merr.Detail != "系统繁忙，请稍后再试" {
		t.Fatalf("internal detail leaked: %q", merr.Detail)
	}
}

func TestToMicroError_Unknown(t *testing.T) {
	merr := asMicroError(t, ToMicroError(errors.New("boom")))
	if merr.Code != HTTPInternalError || merr.Reason != errorx.ErrInternal {
		t.Fatalf("got code=%d reason=%q", merr.Code, merr.Reason)
	}
}

func TestFromMicroError_Nil(t *testing.T) {
	ok, err := FromMicroError(nil)
	if ok || err != nil {
		t.Fatalf("got ok=%v err=%v", ok, err)
	}
}

func TestFromMicroError_Reason(t *testing.T) {
	remote := &microerrors.Error{Id: "svc", Code: HTTPBadRequest, Detail: "用户不存在", Reason: "USER_NOT_FOUND"}
	ok, err := FromMicroError(remote)
	if !ok {
		t.Fatal("expected restored")
	}
	biz, isBiz := err.(errorx.BizError)
	if !isBiz {
		t.Fatalf("expected errorx.BizError, got %T", err)
	}
	if biz.CodeStr() != "USER_NOT_FOUND" || biz.MsgStr() != "用户不存在" {
		t.Fatalf("got code=%q msg=%q", biz.CodeStr(), biz.MsgStr())
	}
}

func TestFromMicroError_CodeFallback(t *testing.T) {
	cases := []struct {
		code int32
		want string
	}{
		{HTTPBadRequest, errorx.ErrParam},
		{HTTPUnauthorized, errorx.ErrUnauthorized},
		{HTTPInternalError, errorx.ErrInternal},
	}
	for _, c := range cases {
		remote := &microerrors.Error{Id: "svc", Code: c.code, Detail: "d"}
		ok, err := FromMicroError(remote)
		if !ok {
			t.Fatalf("code=%d: expected restored", c.code)
		}
		biz, isBiz := err.(errorx.BizError)
		if !isBiz || biz.CodeStr() != c.want {
			t.Fatalf("code=%d: got %T %v, want reason %q", c.code, err, err, c.want)
		}
	}
}

func TestFromMicroError_Plain(t *testing.T) {
	plain := errors.New("plain")
	ok, err := FromMicroError(plain)
	if ok || err != plain {
		t.Fatalf("got ok=%v err=%v", ok, err)
	}
}
