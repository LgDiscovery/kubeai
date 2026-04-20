// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package model

import (
	"context"
	"kubeai-model-manager/pkg/metrics"

	"kubeai-model-manager/internal/svc"
	"kubeai-model-manager/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteModelLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteModelLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteModelLogic {
	return &DeleteModelLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteModelLogic) DeleteModel(name string) (resp *types.CommonResp, err error) {
	m, err := l.svcCtx.ModelRepo.GetByName(l.ctx, name)
	if err != nil {
		return nil, err
	}
	versions, err := l.svcCtx.VersionRepo.ListByModel(l.ctx, m.ID)
	if err != nil {
		return nil, err
	}
	for _, version := range versions {
		err = l.svcCtx.VersionRepo.Delete(l.ctx, version.ID)
		if err != nil {
			return nil, err
		}
		err = l.svcCtx.MinIOClient.Delete(l.ctx, version.StoragePath)
		if err != nil {
			return nil, err
		}
	}
	err = l.svcCtx.ModelRepo.Delete(l.ctx, m.ID)
	if err != nil {
		return nil, err
	}
	resp = &types.CommonResp{
		Code:    0,
		Message: "success",
	}
	metrics.ModelTotal.Dec()
	return resp, nil
}
