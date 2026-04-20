// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package model

import (
	"context"

	"kubeai-model-manager/internal/svc"
	"kubeai-model-manager/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetModelLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetModelLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetModelLogic {
	return &GetModelLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetModelLogic) GetModel(name string) (resp *types.Model, err error) {
	m, err := l.svcCtx.ModelRepo.GetByName(l.ctx, name)
	if err != nil {
		return nil, err
	}
	resp = toModelResponse(m)
	return resp, nil
}
