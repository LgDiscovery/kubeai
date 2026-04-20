// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package job_scheduler

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"kubeai-job-scheduler/internal/help"
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

// ProcessTrainingTask 处理推理任务（由消费者调用）
func (l *SubmitTrainingTaskLogic) ProcessTrainingTask(taskID string, data []byte) error {
	task, err := model.UnmarshalTrainingTask(data)
	if err != nil {
		return err
	}
	logx.Infof("processing training task %s", task.TaskID)

	resourceReq := help.ConvertToResourceRequest(task.Resources)
	nodeName, err := l.svcCtx.ResourceTracker.FindFitNode(resourceReq, l.svcCtx.PlacementStrategy)
	if err != nil {
		return l.handleTrainingTaskFailure(task, err)
	}
	podName, err := l.createTrainingJob(task, nodeName)
	if err != nil {
		return l.handleTrainingTaskFailure(task, err)
	}
	task.PodName = podName
	task.Status = model.StatusRunning
	task.UpdatedAt = time.Now()
	return l.saveTrainingTaskState(task)
}

func (l *SubmitTrainingTaskLogic) handleTrainingTaskFailure(task *model.TrainingTask, err error) error {
	task.RetryCount++
	task.ErrorMessage = err.Error()
	task.UpdatedAt = time.Now()
	if task.RetryCount >= task.MaxRetries {
		task.Status = model.StatusFailed
		// 4. 移入死信队列
		data, _ := task.Marshal()
		if err := l.svcCtx.DeadLetterQueue.Push(l.ctx, task.TaskID, data, err.Error()); err != nil {
			return fmt.Errorf("dead letter queue push failed, %v", err)
		}
	} else {
		task.Status = model.StatusPending
		// 重新入队（延迟）
		time.Sleep(time.Duration(task.RetryCount) * time.Second)
		data, _ := task.Marshal()
		if err := l.svcCtx.TrainingQueue.Push(l.ctx, task.TaskID, data, task.Priority); err != nil {
			return fmt.Errorf("training task push failed, %v", err)
		}
	}
	return l.saveTrainingTaskState(task)
}

func (l *SubmitTrainingTaskLogic) saveTrainingTaskState(task *model.TrainingTask) error {
	key := fmt.Sprintf("kubeai:task:training:%s", task.TaskID)
	data, err := task.Marshal()
	if err != nil {
		return fmt.Errorf("marshal training task failed, %v", err)
	}
	return l.svcCtx.RedisClient.Set(l.ctx, key, data, 24*time.Hour).Err()
}

func (l *SubmitTrainingTaskLogic) createTrainingJob(task *model.TrainingTask, name string) (string, error) {
	return "", nil
}
