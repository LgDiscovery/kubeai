// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package health

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"kubeai-model-manager/internal/logic/health"
	"kubeai-model-manager/internal/svc"
)

func HealthHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := health.NewHealthLogic(r.Context(), svcCtx)
		resp, err := l.Health()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
