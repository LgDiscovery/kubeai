// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package inference

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"kubeai-inference-gateway/internal/logic/inference"
	"kubeai-inference-gateway/internal/svc"
)

func GetTrainingHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := inference.NewGetTrainingLogic(r.Context(), svcCtx)
		resp, err := l.GetTraining()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
