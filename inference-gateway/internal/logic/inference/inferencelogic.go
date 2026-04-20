// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package inference

import (
	"context"

	"kubeai-inference-gateway/internal/svc"
	"kubeai-inference-gateway/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type InferenceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewInferenceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InferenceLogic {
	return &InferenceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *InferenceLogic) Inference(req *types.InferenceRequest) (resp *types.InferenceResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
