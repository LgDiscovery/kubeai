// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package inference

import (
	"context"
	"fmt"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"kubeai-inference-gateway/internal/svc"
	"kubeai-inference-gateway/internal/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/zeromicro/go-zero/core/logx"
	aiv1 "kubeai-inference-gateway/trainingjob/api/v1"
)

type PauseTrainingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPauseTrainingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PauseTrainingLogic {
	return &PauseTrainingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PauseTrainingLogic) PauseTraining(req *types.ControlReq) (resp *types.CommonResp, err error) {
	jobName := fmt.Sprintf("rain%-%s", req.TaskID)
	job := &aiv1.TrainingJob{}
	if err := l.svcCtx.CtrlClient.Get(l.ctx, k8sTypes.NamespacedName{Name: jobName, Namespace: l.svcCtx.Config.K8s.Namespace}, job); err != nil {
		return nil, err
	}
	patch := client.RawPatch(k8sTypes.MergePatchType, []byte(`{"spec":{"runPolicy":{"suspend":true}}}`))
	if err := l.svcCtx.CtrlClient.Patch(l.ctx, job, patch); err != nil {
		return nil, err
	}
	return &types.CommonResp{
		Code:    0,
		Message: "pause training success",
	}, nil
}
