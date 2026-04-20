// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package model

import (
	"context"

	"kubeai-model-manager/internal/svc"
	"kubeai-model-manager/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListModelsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListModelsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListModelsLogic {
	return &ListModelsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListModelsLogic) ListModels(req *types.ModelListReq) (resp *types.ModelListResp, err error) {
	offset := (req.Page - 1) * req.PageSize
	ms, total, err := l.svcCtx.ModelRepo.List(l.ctx, offset, req.PageSize, req.Framework, req.TaskType)
	if err != nil {
		return nil, err
	}
	resp = &types.ModelListResp{
		Items:    make([]types.Model, 0, len(ms)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	for _, m := range ms {
		resp.Items = append(resp.Items, *toModelResponse(&m))
	}
	return resp, nil
}
