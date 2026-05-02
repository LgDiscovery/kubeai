// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package model

import (
	"kubeai-model-manager/internal/types"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"kubeai-model-manager/internal/logic/model"
	"kubeai-model-manager/internal/svc"
)

func GetMetadataHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CommonReq
		if err := httpx.ParseForm(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		l := model.NewGetMetadataLogic(r.Context(), svcCtx)
		resp, err := l.GetMetadata(req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
