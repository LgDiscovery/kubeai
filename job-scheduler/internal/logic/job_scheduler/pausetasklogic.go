// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package job_scheduler

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubeai-job-scheduler/internal/svc"
	"kubeai-job-scheduler/internal/types"

	"github.com/zeromicro/go-zero/core/logx"

	k8stypes "k8s.io/apimachinery/pkg/types"
)

type PauseTaskLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPauseTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PauseTaskLogic {
	return &PauseTaskLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PauseTaskLogic) PauseTask(req *types.TaskControlReq) (resp *types.CommonResp, err error) {
	if req.TaskType == "inference" {
		podName := fmt.Sprintf("inference-%s", req.TaskID)
		patch := []byte(`{"metadata":{"annotations":{"kubeai.io/paused":"true"}}}`)
		_, err := l.svcCtx.K8sClient.CoreV1().Pods(l.svcCtx.Config.K8s.Namespace).Patch(
			l.ctx, podName, k8stypes.StrategicMergePatchType, patch, metav1.PatchOptions{})
		if err != nil {
			return nil, err
		}
		return &types.CommonResp{
			Code:    0,
			Message: "pause task success",
			Data:    nil,
		}, nil
	}
	return &types.CommonResp{
		Code:    0,
		Message: "pause task success",
		Data:    nil,
	}, nil
}
