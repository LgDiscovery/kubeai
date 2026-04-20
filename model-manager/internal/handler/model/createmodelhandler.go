// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package model

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"kubeai-model-manager/internal/logic/model"
	"kubeai-model-manager/internal/svc"
	"kubeai-model-manager/internal/types"
)

func CreateModelHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CreateModelReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 从上下文获取 owner（网关透传）
		if owner := r.Header.Get("X-User-ID"); owner != "" {
			req.Owner = owner
		}

		l := model.NewCreateModelLogic(r.Context(), svcCtx)
		resp, err := l.CreateModel(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
