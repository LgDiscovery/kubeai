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

type UpdateVersionStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateVersionStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateVersionStatusLogic {
	return &UpdateVersionStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateVersionStatusLogic) UpdateVersionStatus(req *types.UpdateVersionStatusReq) (resp *types.CommonResp, err error) {
	m, err := l.svcCtx.ModelRepo.GetByName(l.ctx, req.Name)
	if err != nil {
		return nil, err
	}
	v, err := l.svcCtx.VersionRepo.GetByModelAndVersion(l.ctx, m.ID, req.Version)
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.VersionRepo.UpdateStatus(l.ctx, v.ID, req.Status); err != nil {
		return nil, err
	}

	if req.Status == "active" {
		metrics.ModelHealthStatus.WithLabelValues(req.Name, req.Version).Set(1)
	} else {
		metrics.ModelHealthStatus.WithLabelValues(req.Name, req.Version).Set(0)
		metrics.ModelVersionTotal.WithLabelValues("active").Dec()
	}
	return &types.CommonResp{
		Code:    0,
		Message: "success",
	}, nil
}
