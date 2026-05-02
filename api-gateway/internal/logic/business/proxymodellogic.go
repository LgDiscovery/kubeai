// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package business

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"kubeai-api-gateway/internal/svc"
)

type ProxyModelLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProxyModelLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ProxyModelLogic {
	return &ProxyModelLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProxyModelLogic) ProxyModel() error {
	// todo: add your logic here and delete this line

	return nil
}
