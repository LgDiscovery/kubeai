// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package business

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"kubeai-api-gateway/internal/svc"
)

type ProxyInferenceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProxyInferenceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ProxyInferenceLogic {
	return &ProxyInferenceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProxyInferenceLogic) ProxyInference() error {
	// todo: add your logic here and delete this line

	return nil
}
