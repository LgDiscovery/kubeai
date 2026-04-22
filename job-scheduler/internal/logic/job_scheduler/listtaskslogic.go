// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package job_scheduler

import (
	"context"

	"kubeai-job-scheduler/internal/svc"
	"kubeai-job-scheduler/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListTasksLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListTasksLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTasksLogic {
	return &ListTasksLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListTasksLogic) ListTasks(req *types.ListTasksReq) (resp *types.ListTasksResp, err error) {
	offset := (req.Page - 1) * req.PageSize
	if req.TaskType == "inference" {
		tasks, total, err := l.svcCtx.InferenceTaskRepo.List(l.ctx, req.Status, offset, req.PageSize)
		if err != nil {
			return nil, err
		}
		items := make([]interface{}, 0, len(tasks))
		for _, task := range tasks {
			items = append(items, task)
		}
		resp = &types.ListTasksResp{
			Items:    items,
			Total:    total,
			Page:     req.Page,
			PageSize: req.PageSize,
		}
		return
	} else if req.TaskType == "training" {
		tasks, total, err := l.svcCtx.TrainingTaskRepo.List(l.ctx, req.Status, offset, req.PageSize)

		if err != nil {
			return nil, err
		}
		items := make([]interface{}, 0, len(tasks))
		for _, task := range tasks {
			items = append(items, task)
		}
		resp = &types.ListTasksResp{
			Items:    items,
			Total:    total,
			Page:     req.Page,
			PageSize: req.PageSize,
		}
		return
	} else {
		return nil, nil
	}
}
