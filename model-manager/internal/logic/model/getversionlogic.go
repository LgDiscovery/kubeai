// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package model

import (
	"context"

	"kubeai-model-manager/internal/svc"
	"kubeai-model-manager/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetVersionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetVersionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetVersionLogic {
	return &GetVersionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetVersionLogic) GetVersion(req types.CommonReq) (resp *types.ModelVersion, err error) {
	m, err := l.svcCtx.ModelRepo.GetByName(l.ctx, req.Name)
	if err != nil {
		l.Logger.Errorf("GetVersion err: %v", err)
		return nil, err
	}
	v, err := l.svcCtx.VersionRepo.GetByModelAndVersion(l.ctx, m.ID, req.Version)
	if err != nil {
		l.Logger.Errorf("GetVersion err: %v", err)
		return nil, err
	}
	return toVersionResponse(v), nil

}
