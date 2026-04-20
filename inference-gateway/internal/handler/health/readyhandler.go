// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package health

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"kubeai-inference-gateway/internal/logic/health"
	"kubeai-inference-gateway/internal/svc"
)

func ReadyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := health.NewReadyLogic(r.Context(), svcCtx)
		resp, err := l.Ready()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
