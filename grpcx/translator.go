package grpcx

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gocrud/pkg/errorx"
	"github.com/gocrud/veri"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	HeaderBizCode   = "x-biz-code"
	HeaderBizReason = "x-biz-reason"
	HeaderBizDetail = "x-biz-detail"
)

type bizErrorSniffer interface {
	CodeStr() string
	MsgStr() string
}

func ToGRPCError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	var valErrs *veri.ValidationErrors
	if errors.As(err, &valErrs) {
		var detailBuilder strings.Builder
		for index, fieldErr := range valErrs.Errors {
			if index > 0 {
				detailBuilder.WriteString("; ")
			}
			detailBuilder.WriteString(fmt.Sprintf("%s:%s", fieldErr.Field, fieldErr.Message))
		}
		trailer := metadata.Pairs(
			HeaderBizCode, errorx.ErrParam,
			HeaderBizReason, "请求参数校验失败",
			HeaderBizDetail, detailBuilder.String(),
		)
		_ = grpc.SetTrailer(ctx, trailer)
		return status.Error(codes.InvalidArgument, "参数校验失败")
	}
	var biz bizErrorSniffer
	if errors.As(err, &biz) && biz.CodeStr() != errorx.ErrInternal {
		trailer := metadata.Pairs(HeaderBizCode, biz.CodeStr(), HeaderBizReason, biz.MsgStr())
		_ = grpc.SetTrailer(ctx, trailer)
		return status.Error(codes.Aborted, biz.MsgStr())
	}
	trailer := metadata.Pairs(HeaderBizCode, errorx.ErrInternal, HeaderBizReason, "系统繁忙，请稍后再试")
	_ = grpc.SetTrailer(ctx, trailer)
	return status.Error(codes.Internal, "系统内部错误")
}

func FromGRPCError(err error, trailer metadata.MD) (bool, error) {
	if err == nil {
		return false, nil
	}
	result, ok := status.FromError(err)
	if !ok {
		return false, err
	}
	if result.Code() == codes.Aborted || result.Code() == codes.Internal || result.Code() == codes.InvalidArgument {
		var bizCode, bizReason string
		if values := trailer.Get(HeaderBizCode); len(values) > 0 {
			bizCode = values[0]
		}
		if reasons := trailer.Get(HeaderBizReason); len(reasons) > 0 {
			bizReason = reasons[0]
		}
		if bizCode != "" {
			return true, errorx.E(bizCode, bizReason)
		}
	}
	return false, err
}
