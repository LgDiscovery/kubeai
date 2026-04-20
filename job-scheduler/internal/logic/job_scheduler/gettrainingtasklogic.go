// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package job_scheduler

import (
	"context"

	"kubeai-job-scheduler/internal/svc"
	"kubeai-job-scheduler/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTrainingTaskLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetTrainingTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTrainingTaskLogic {
	return &GetTrainingTaskLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTrainingTaskLogic) GetTrainingTask(req *types.GetTrainingTaskReq) (resp *types.TrainingTask, err error) {
	// todo: add your logic here and delete this line

	return
}
