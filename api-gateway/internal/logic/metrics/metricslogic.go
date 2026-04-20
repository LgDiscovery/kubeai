// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package metrics

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"kubeai-api-gateway/internal/svc"
)

type MetricsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMetricsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MetricsLogic {
	return &MetricsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MetricsLogic) Metrics() error {
	return nil
}
