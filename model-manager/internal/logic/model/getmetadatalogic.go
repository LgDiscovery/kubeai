// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package model

import (
	"context"
	"kubeai-model-manager/internal/model"

	"kubeai-model-manager/internal/svc"
	"kubeai-model-manager/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMetadataLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMetadataLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMetadataLogic {
	return &GetMetadataLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMetadataLogic) GetMetadata(req types.CommonReq) (resp *types.ModelMetadataResp, err error) {
	m, err := l.svcCtx.ModelRepo.GetByName(l.ctx, req.Name)
	if err != nil {
		l.Logger.Errorf("GetModelByName err: %v", err)
		return nil, err
	}
	v, err := l.svcCtx.VersionRepo.GetByModelAndVersion(l.ctx, m.ID, req.Version)
	if err != nil {
		l.Logger.Errorf("GetModelVersionByModelAndVersion err: %v", err)
		return nil, err
	}
	return toMetadataResponse(v), nil
}

func toMetadataResponse(v *model.ModelVersion) *types.ModelMetadataResp {
	return &types.ModelMetadataResp{
		ModelName:    v.Model.Name,
		ModelVersion: v.Version,
		Framework:    v.Framework,
		StoragePath:  v.StoragePath,
	}
}
