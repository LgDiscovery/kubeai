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

func UpdateVersionStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdateVersionStatusReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := model.NewUpdateVersionStatusLogic(r.Context(), svcCtx)
		resp, err := l.UpdateVersionStatus(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
