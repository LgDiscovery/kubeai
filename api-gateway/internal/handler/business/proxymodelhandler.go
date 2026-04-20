// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package business

import (
	"kubeai-api-gateway/internal/logic"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"kubeai-api-gateway/internal/svc"
)

func ProxyModelHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewProxyLogic(r.Context(), svcCtx)
		if err := l.ProxyRequest(w, r); err != nil {
			httpx.Error(w, err)
		}
	}
}
