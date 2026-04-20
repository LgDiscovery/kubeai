package repo

import (
	"context"

	"gorm.io/gorm"
	"kubeai-model-manager/internal/model"
)

type ModelRepo struct {
	db *gorm.DB
}

func NewModelRepo(db *gorm.DB) *ModelRepo {
	return &ModelRepo{db: db}
}

func (r *ModelRepo) Create(ctx context.Context, m *model.Model) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *ModelRepo) GetByID(ctx context.Context, id uint) (*model.Model, error) {
	var m model.Model
	err := r.db.WithContext(ctx).Preload("Versions").First(&m, id).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *ModelRepo) GetByName(ctx context.Context, name string) (*model.Model, error) {
	var m model.Model
	err := r.db.WithContext(ctx).Where("name = ?", name).Preload("Versions").First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *ModelRepo) List(ctx context.Context, offset, limit int, framework, taskType string) ([]model.Model, int64, error) {
	var models []model.Model
	var total int64
	query := r.db.WithContext(ctx).Model(&model.Model{})
	if framework != "" {
		query = query.Where("framework = ?", framework)
	}
	if taskType != "" {
		query = query.Where("task_type = ?", taskType)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&models).Error
	return models, total, err
}

func (r *ModelRepo) Update(ctx context.Context, m *model.Model) error {
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *ModelRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Model{}, id).Error
}
