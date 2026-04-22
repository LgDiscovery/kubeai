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
	ctx       context.Context
	svcCtx    *svc.ServiceContext
	streamKey string
}

func NewSubmitTrainingTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SubmitTrainingTaskLogic {
	return &SubmitTrainingTaskLogic{
		Logger:    logx.WithContext(ctx),
		ctx:       ctx,
		svcCtx:    svcCtx,
		streamKey: svcCtx.Config.Redis.Streams.Training,
	}
}

func (l *SubmitTrainingTaskLogic) SubmitTraining(req *types.SubmitTrainingReq) (resp *types.TrainingTaskResp, err error) {
	var env []model.EnvVar
	if req.Env != nil {
		for _, v := range req.Env {
			env = append(env, model.EnvVar{
				Name:  v.Name,
				Value: v.Value,
			})
		}
	}
	task := &model.TrainingTask{
		TaskID:      "train-" + uuid.New().String()[:8],
		Name:        req.Name,
		ModelName:   req.ModelName,
		Framework:   req.Framework,
		Image:       req.Image,
		Command:     req.Command,
		Args:        req.Args,
		Resources:   model.ResourceRequest(req.Resources),
		Distributed: req.Distributed,
		WorkerNum:   req.WorkerNum,
		Env:         env,
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

	// 保存任务到数据库
	if err := l.svcCtx.TrainingTaskRepo.Create(l.ctx, task); err != nil {
		return nil, fmt.Errorf("save training task failed, %v", err)
	}
	return &types.TrainingTaskResp{
		TaskID:  task.TaskID,
		Status:  string(model.StatusPending),
		Message: "training task submitted successfully",
	}, nil
}
