// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package model

import (
	"context"

	"kubeai-model-manager/internal/svc"
	"kubeai-model-manager/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateModelLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateModelLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateModelLogic {
	return &UpdateModelLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateModelLogic) UpdateModel(req *types.UpdateModelReq) (resp *types.Model, err error) {
	m, err := l.svcCtx.ModelRepo.GetByName(l.ctx, req.Name)
	if err != nil {
		return nil, err
	}
	if req.Description != m.Description && req.Description != "" {
		m.Description = req.Description
	}
	if req.Labels != m.Labels && req.Labels != "" {
		m.Labels = req.Labels
	}
	err = l.svcCtx.ModelRepo.Update(l.ctx, m)
	if err != nil {
		return nil, err
	}
	resp = toModelResponse(m)
	return resp, nil
}
