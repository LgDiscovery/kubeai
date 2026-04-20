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

func GetModelHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := model.NewGetModelLogic(r.Context(), svcCtx)
		var req types.CommonReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := l.GetModel(req.Name)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
