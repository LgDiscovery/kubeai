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
)

type CancelTaskLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCancelTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelTaskLogic {
	return &CancelTaskLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CancelTaskLogic) CancelTask(req *types.TaskControlReq) (resp *types.CommonResp, err error) {
	if req.TaskType == "inference" {
		podName := fmt.Sprintf("inference-%s", req.TaskID)
		err = l.svcCtx.K8sClient.CoreV1().Pods(l.svcCtx.Config.K8s.Namespace).Delete(
			l.ctx, podName, metav1.DeleteOptions{})
		if err != nil {
			return nil, err
		}
		return &types.CommonResp{
			Code:    0,
			Message: "cancel task success",
			Data:    nil,
		}, nil
	}
	return &types.CommonResp{
		Code:    0,
		Message: "cancel task success",
		Data:    nil,
	}, nil
}
