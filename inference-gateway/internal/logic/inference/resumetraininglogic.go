// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package inference

import (
	"context"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	aiv1 "kubeai-inference-gateway/trainingjob/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"kubeai-inference-gateway/internal/svc"
	"kubeai-inference-gateway/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ResumeTrainingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewResumeTrainingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResumeTrainingLogic {
	return &ResumeTrainingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ResumeTrainingLogic) ResumeTraining(req *types.ControlReq) (resp *types.CommonResp, err error) {
	jobName := req.TaskID
	job := &aiv1.TrainingJob{}
	if err := l.svcCtx.CtrlClient.Get(l.ctx, k8sTypes.NamespacedName{Name: jobName, Namespace: l.svcCtx.Config.K8s.Namespace}, job); err != nil {
		return nil, err
	}
	patch := client.RawPatch(k8sTypes.MergePatchType, []byte(`{"spec":{"runPolicy":{"suspend":false}}}`))
	if err := l.svcCtx.CtrlClient.Patch(l.ctx, job, patch); err != nil {
		return nil, err
	}
	return &types.CommonResp{
		Code:    0,
		Message: "resume training success",
	}, nil
}
