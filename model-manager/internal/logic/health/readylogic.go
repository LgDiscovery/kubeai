// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package health

import (
	"context"

	"kubeai-model-manager/internal/svc"
	"kubeai-model-manager/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReadyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewReadyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadyLogic {
	return &ReadyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ReadyLogic) Ready() (resp *types.CommonResp, err error) {
	resp = &types.CommonResp{
		Code:    0,
		Message: "success",
	}
	return
}
