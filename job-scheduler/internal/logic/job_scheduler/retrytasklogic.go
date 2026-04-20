// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package job_scheduler

import (
	"context"
	"fmt"
	"kubeai-job-scheduler/internal/model"
	"time"

	"kubeai-job-scheduler/internal/svc"
	"kubeai-job-scheduler/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RetryTaskLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRetryTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RetryTaskLogic {
	return &RetryTaskLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RetryTaskLogic) RetryTask(req *types.TaskControlReq) (resp *types.CommonResp, err error) {
	var key string
	if req.TaskType == "inference" {
		key = fmt.Sprintf("%s:%s", l.svcCtx.Config.Redis.Streams.Inference, req.TaskID)
	} else {
		key = fmt.Sprintf("%s:%s", l.svcCtx.Config.Redis.Streams.Training, req.TaskID)
	}
	data, err := l.svcCtx.RedisClient.Get(l.ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	if req.TaskType == "inference" {
		task, err := model.UnmarshalInferenceTask(data)
		if err != nil {
			return nil, err
		}
		task.Status = model.StatusPending
		task.RetryCount = 0
		task.UpdatedAt = time.Now()
		newData, _ := task.Marshal()
		err = l.svcCtx.InferenceQueue.Push(l.ctx, task.TaskID, newData, task.Priority)
		if err != nil {
			return nil, err
		}
		return &types.CommonResp{
			Code:    0,
			Message: "retry task success",
			Data:    nil,
		}, nil
	} else {
		task, err := model.UnmarshalInferenceTask(data)
		if err != nil {
			return nil, err
		}
		task.Status = model.StatusPending
		task.RetryCount = 0
		task.UpdatedAt = time.Now()
		newData, _ := task.Marshal()
		err = l.svcCtx.TrainingQueue.Push(l.ctx, task.TaskID, newData, task.Priority)
		if err != nil {
			return nil, err
		}
		return &types.CommonResp{
			Code:    0,
			Message: "retry task success",
			Data:    nil,
		}, nil
	}
}
