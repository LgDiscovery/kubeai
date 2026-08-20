// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package health

import (
	"context"
	"time"

	"kubeai-inference-gateway/internal/svc"
	"kubeai-inference-gateway/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReadyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewReadyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadyLogic {
	return &ReadyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Ready 就绪检查（readiness）
func (l *ReadyLogic) Ready() (resp *types.CommonResp, err error) {
	healthLogic := NewHealthLogic(l.ctx, l.svcCtx)
	deps := healthLogic.CheckDependencies()

	allReady := true
	for _, dep := range deps {
		if depMap, ok := dep.(map[string]interface{}); ok {
			if status, exists := depMap["status"]; exists && status != "UP" {
				allReady = false
				break
			}
		}
	}

	status := "READY"
	code := 0
	message := "service is ready"
	if !allReady {
		status = "NOT_READY"
		code = 1
		message = "some dependencies are not ready"
	}

	resp = &types.CommonResp{
		Code:    code,
		Message: message,
		Data: map[string]interface{}{
			"status":       status,
			"service":      "inference-gateway",
			"timestamp":    time.Now().Format(time.RFC3339),
			"dependencies": deps,
		},
	}
	return
}
