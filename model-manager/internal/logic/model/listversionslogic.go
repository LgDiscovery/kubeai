// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package model

import (
	"context"

	"kubeai-model-manager/internal/svc"
	"kubeai-model-manager/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListVersionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListVersionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListVersionsLogic {
	return &ListVersionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListVersionsLogic) ListVersions(req *types.CommonReq) (resp *types.ModelVersionListResp, err error) {
	m, err := l.svcCtx.ModelRepo.GetByName(l.ctx, req.Name)
	if err != nil {
		return nil, err
	}
	versions, err := l.svcCtx.VersionRepo.ListByModel(l.ctx, m.ID)
	if err != nil {
		return nil, err
	}
	items := make([]types.ModelVersion, len(versions))
	for i, v := range versions {
		items[i] = *toVersionResponse(&v)
	}
	return &types.ModelVersionListResp{
		Items: items,
	}, nil
}
