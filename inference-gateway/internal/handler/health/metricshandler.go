// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package health

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"kubeai-inference-gateway/internal/svc"
)

func MetricsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svcCtx.Config.Metrics.Enabled {
			promhttp.Handler().ServeHTTP(w, r)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}
}
