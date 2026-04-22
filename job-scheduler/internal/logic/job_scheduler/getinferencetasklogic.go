// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package job_scheduler

import (
	"context"
	"kubeai-job-scheduler/internal/model"

	"kubeai-job-scheduler/internal/svc"
	"kubeai-job-scheduler/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetInferenceTaskLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetInferenceTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetInferenceTaskLogic {
	return &GetInferenceTaskLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetInferenceTaskLogic) GetInferenceTask(req *types.GetInferenceTaskReq) (resp *model.InferenceTask, err error) {
	return l.svcCtx.InferenceTaskRepo.GetByTaskID(l.ctx, req.TaskID)
}
