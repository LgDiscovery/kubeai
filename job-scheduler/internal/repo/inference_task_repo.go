package repo

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"kubeai-job-scheduler/internal/model"
)

type InferenceTaskRepo struct {
	db *gorm.DB
}

func NewInferenceTaskRepo(db *gorm.DB) *InferenceTaskRepo {
	return &InferenceTaskRepo{
		db: db,
	}
}

func (r *InferenceTaskRepo) Create(ctx context.Context, task *model.InferenceTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *InferenceTaskRepo) Update(ctx context.Context, task *model.InferenceTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

func (r *InferenceTaskRepo) Delete(ctx context.Context, task *model.InferenceTask) error {
	return r.db.WithContext(ctx).Delete(task).Error
}

func (r *InferenceTaskRepo) GetByTaskID(ctx context.Context, taskID string) (*model.InferenceTask, error) {
	var task model.InferenceTask
	err := r.db.WithContext(ctx).Where("task_id = ?", taskID).First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &task, err
}

func (r *InferenceTaskRepo) List(ctx context.Context, status string, offset, limit int) ([]*model.InferenceTask, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.InferenceTask{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var tasks []*model.InferenceTask
	if err := query.Order("created_at desc").Limit(limit).Offset(offset).Find(&tasks).Error; err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}
