// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package inference

import (
	"context"

	"kubeai-inference-gateway/internal/svc"
	"kubeai-inference-gateway/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type TrainingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewTrainingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TrainingLogic {
	return &TrainingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TrainingLogic) Training(req *types.TrainingTaskReq) (resp *types.TrainingTaskResp, err error) {
	// todo: add your logic here and delete this line

	return
}
