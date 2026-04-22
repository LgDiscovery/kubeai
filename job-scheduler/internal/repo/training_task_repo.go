package repo

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"kubeai-job-scheduler/internal/model"
)

type TrainingTaskRepo struct {
	db *gorm.DB
}

func NewTrainingTaskRepo(db *gorm.DB) *TrainingTaskRepo {
	return &TrainingTaskRepo{db: db}
}

func (r *TrainingTaskRepo) Create(ctx context.Context, task *model.TrainingTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *TrainingTaskRepo) Update(ctx context.Context, task *model.TrainingTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

func (r *TrainingTaskRepo) GetByTaskID(ctx context.Context, taskID string) (*model.TrainingTask, error) {
	var task model.TrainingTask
	err := r.db.WithContext(ctx).Where("task_id = ?", taskID).First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &task, err
}

func (r *TrainingTaskRepo) List(ctx context.Context, status string, offset, limit int) ([]*model.TrainingTask, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.TrainingTask{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var tasks []*model.TrainingTask
	err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&tasks).Error
	return tasks, total, err
}

func (r *TrainingTaskRepo) Delete(ctx context.Context, taskID string) error {
	return r.db.WithContext(ctx).Where("task_id = ?", taskID).Delete(&model.TrainingTask{}).Error
}
