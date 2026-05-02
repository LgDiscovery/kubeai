// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package job_scheduler

import (
	"context"
	"fmt"
	"kubeai-job-scheduler/internal/model"

	"kubeai-job-scheduler/internal/svc"
	"kubeai-job-scheduler/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CallBackTaskStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCallBackTaskStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CallBackTaskStatusLogic {
	return &CallBackTaskStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CallBackTaskStatusLogic) CallBackTaskStatus(req *types.CallBackTaskStatusReq) (resp *types.CommonResp, err error) {
	if req.TaskID == "" || req.Name == "" || req.Status == "" {
		return nil, fmt.Errorf("task id, name or status is empty")
	}
	if req.TaskType == "" {
		return nil, fmt.Errorf("task type is empty")
	}
	taskType := model.TaskType(req.TaskType)
	if taskType == model.TrainingTaskType {
		err = l.svcCtx.TrainingTaskRepo.CallbackStatus(l.ctx, req.Name, model.TaskStatus(req.Status))
	} else if taskType == model.InferenceTaskType {
		err = l.svcCtx.InferenceTaskRepo.CallbackStatus(l.ctx, req.Name, model.TaskStatus(req.Status))
	} else {
		return nil, fmt.Errorf("task type %s is not supported", req.TaskType)
	}
	if err != nil {
		return nil, err
	}
	return &types.CommonResp{
		Code:    0,
		Message: "success",
	}, nil
}
