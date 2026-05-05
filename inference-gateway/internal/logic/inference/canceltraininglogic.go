// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package inference

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	aiv1 "kubeai-inference-gateway/trainingjob/api/v1"

	"kubeai-inference-gateway/internal/svc"
	"kubeai-inference-gateway/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelTrainingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCancelTrainingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelTrainingLogic {
	return &CancelTrainingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CancelTrainingLogic) CancelTraining(req *types.ControlReq) (resp *types.CommonResp, err error) {
	jobName := req.TaskID
	job := &aiv1.TrainingJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: l.svcCtx.Config.K8s.Namespace,
		},
	}
	err = l.svcCtx.CtrlClient.Delete(l.ctx, job)
	if err != nil {
		return nil, err
	}
	return &types.CommonResp{
		Code: 0,
	}, nil
}
