// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package inference

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"kubeai-inference-gateway/internal/logic/inference"
	"kubeai-inference-gateway/internal/svc"
	"kubeai-inference-gateway/internal/types"
)

func InferenceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.InferenceRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := inference.NewInferenceLogic(r.Context(), svcCtx)
		resp, err := l.Inference(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
