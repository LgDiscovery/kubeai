// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package health

import (
	"context"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"
	"kubeai-inference-gateway/internal/svc"
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

// MetricsHandler 暴露 Prometheus 指标端点
func MetricsHandler(svc *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svc.Config.Metrics.Enabled {
			// 使用 promhttp.Handler() 处理请求
			promhttp.Handler().ServeHTTP(w, r)
		} else {
			httpx.OkJson(w, nil)
		}
	}
}
