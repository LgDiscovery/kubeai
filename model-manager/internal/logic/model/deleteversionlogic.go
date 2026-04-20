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

type DeleteVersionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteVersionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteVersionLogic {
	return &DeleteVersionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteVersionLogic) DeleteVersion(req types.CommonReq) (resp *types.CommonResp, err error) {
	m, err := l.svcCtx.ModelRepo.GetByName(l.ctx, req.Name)
	if err != nil {
		return nil, err
	}
	v, err := l.svcCtx.VersionRepo.GetByModelAndVersion(l.ctx, m.ID, req.Version)
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.MinIOClient.Delete(l.ctx, v.StoragePath); err != nil {
		return nil, err
	}
	metrics.ModelVersionTotal.WithLabelValues("active").Dec()
	metrics.ModelHealthStatus.WithLabelValues(req.Name, req.Version).Set(0)
	return &types.CommonResp{
		Code:    0,
		Message: "success",
	}, nil
}
