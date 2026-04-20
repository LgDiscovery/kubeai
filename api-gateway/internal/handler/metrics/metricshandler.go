// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package metrics

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"kubeai-api-gateway/internal/svc"
)

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
