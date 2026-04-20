package repo

import (
	"context"
	"gorm.io/gorm"
	"kubeai-model-manager/internal/model"
)

type VersionRepo struct {
	db *gorm.DB
}

func NewVersionRepo(db *gorm.DB) *VersionRepo {
	return &VersionRepo{db: db}
}

func (r *VersionRepo) Create(ctx context.Context, v *model.ModelVersion) error {
	return r.db.WithContext(ctx).Create(v).Error
}

func (r *VersionRepo) GetByModelAndVersion(ctx context.Context, modelID uint, version string) (*model.ModelVersion, error) {
	var v model.ModelVersion
	err := r.db.WithContext(ctx).Where("model_id = ? AND version = ?", modelID, version).First(&v).Error
	return &v, err
}

func (r *VersionRepo) GetByID(ctx context.Context, id uint) (*model.ModelVersion, error) {
	var v model.ModelVersion
	err := r.db.WithContext(ctx).First(&v, id).Error
	return &v, err
}

func (r *VersionRepo) ListByModel(ctx context.Context, modelID uint) ([]model.ModelVersion, error) {
	var versions []model.ModelVersion
	err := r.db.WithContext(ctx).Where("model_id = ?", modelID).Order("created_at DESC").Find(&versions).Error
	return versions, err
}

func (r *VersionRepo) UpdateStatus(ctx context.Context, id uint, status string) error {
	return r.db.WithContext(ctx).Model(&model.ModelVersion{}).Where("id = ?", id).Update("status", status).Error
}

func (r *VersionRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.ModelVersion{}, id).Error
}
