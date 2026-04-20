// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package model

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"kubeai-model-manager/internal/model"
	"kubeai-model-manager/pkg/metrics"
	"time"

	"kubeai-model-manager/internal/svc"
	"kubeai-model-manager/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateModelLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateModelLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateModelLogic {
	return &CreateModelLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateModelLogic) CreateModel(req *types.CreateModelReq) (resp *types.Model, err error) {
	// 检查名称是否已存在
	_, err = l.svcCtx.ModelRepo.GetByName(l.ctx, req.Name)
	if err == nil {
		return nil, errors.New("model name already exists")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	m := &model.Model{
		Name:        req.Name,
		Description: req.Description,
		Framework:   req.Framework,
		TaskType:    req.TaskType,
		Owner:       req.Owner,
		Labels:      req.Labels,
	}
	if err := l.svcCtx.ModelRepo.Create(l.ctx, m); err != nil {
		return nil, err
	}
	// 更新指标
	metrics.ModelTotal.Inc()
	return toModelResponse(m), nil
}

func toModelResponse(m *model.Model) *types.Model {
	versions := make([]types.ModelVersion, len(m.Versions))
	for i, v := range m.Versions {
		versions[i] = types.ModelVersion{
			ID:           v.ID,
			ModelID:      v.ModelID,
			Version:      v.Version,
			Description:  v.Description,
			StoragePath:  v.StoragePath,
			Framework:    v.Framework,
			FrameworkVer: v.FrameworkVer,
			Metrics:      v.Metrics,
			Parameters:   v.Parameters,
			Size:         v.Size,
			Checksum:     v.Checksum,
			Status:       v.Status,
			CreatedAt:    v.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    v.UpdatedAt.Format(time.RFC3339),
		}
	}
	return &types.Model{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		Framework:   m.Framework,
		TaskType:    m.TaskType,
		Owner:       m.Owner,
		Labels:      m.Labels,
		CreatedAt:   m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   m.UpdatedAt.Format(time.RFC3339),
		Versions:    versions,
	}
}
