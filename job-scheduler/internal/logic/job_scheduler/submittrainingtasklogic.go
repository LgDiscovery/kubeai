// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package job_scheduler

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"kubeai-job-scheduler/internal/model"
	"time"

	"kubeai-job-scheduler/internal/svc"
	"kubeai-job-scheduler/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SubmitTrainingTaskLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSubmitTrainingTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SubmitTrainingTaskLogic {
	return &SubmitTrainingTaskLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SubmitTrainingTaskLogic) SubmitTraining(req *types.SubmitTrainingReq) (resp *types.TrainingTaskResp, err error) {
	task := &model.TrainingTask{
		TaskID:      "train-" + uuid.New().String()[:8],
		Name:        req.Name,
		ModelName:   req.ModelName,
		Framework:   req.Framework,
		Command:     req.Command,
		Args:        req.Args,
		Resources:   model.ResourceRequest(req.Resources),
		DatasetPath: req.DatasetPath,
		OutputPath:  req.OutputPath,
		Status:      model.StatusPending,
		Priority:    req.Priority,
		MaxRetries:  l.svcCtx.Config.Redis.MaxRetries,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	data, _ := task.Marshal()
	if err := l.svcCtx.TrainingQueue.Push(l.ctx, task.TaskID, data, task.Priority); err != nil {
		return nil, err
	}
	if err := l.saveTrainingTaskState(task); err != nil {
		return nil, fmt.Errorf("save training task state failed, %v", err)
	}
	return &types.TrainingTaskResp{
		TaskID:  task.TaskID,
		Status:  string(model.StatusPending),
		Message: "training task submitted successfully",
	}, nil
}

func (l *SubmitTrainingTaskLogic) saveTrainingTaskState(task *model.TrainingTask) error {
	key := fmt.Sprintf("kubeai:task:training:%s", task.TaskID)
	data, err := task.Marshal()
	if err != nil {
		return fmt.Errorf("marshal training task failed, %v", err)
	}
	return l.svcCtx.RedisClient.Set(l.ctx, key, data, 24*time.Hour).Err()
}
