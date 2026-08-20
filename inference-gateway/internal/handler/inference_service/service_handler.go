package inference_service

import (
	"errors"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"kubeai-inference-gateway/internal/logic/inference_service"
	"kubeai-inference-gateway/internal/svc"
	"kubeai-inference-gateway/internal/types"
)

// ListInferenceServiceHandler 列出推理服务
func ListInferenceServiceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := inference_service.NewListInferenceServiceLogic(r.Context(), svcCtx)
		resp, err := l.ListInferenceService()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

// GetInferenceServiceHandler 获取推理服务详情
func GetInferenceServiceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if name == "" {
			httpx.ErrorCtx(r.Context(), w, errors.New("name is required"))
			return
		}
		l := inference_service.NewGetInferenceServiceLogic(r.Context(), svcCtx)
		resp, err := l.GetInferenceService(name)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

// CreateInferenceServiceHandler 创建推理服务
func CreateInferenceServiceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CreateInferenceServiceReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		l := inference_service.NewCreateInferenceServiceLogic(r.Context(), svcCtx)
		resp, err := l.CreateInferenceService(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

// DeleteInferenceServiceHandler 删除推理服务
func DeleteInferenceServiceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if name == "" {
			httpx.ErrorCtx(r.Context(), w, errors.New("name is required"))
			return
		}
		l := inference_service.NewDeleteInferenceServiceLogic(r.Context(), svcCtx)
		resp, err := l.DeleteInferenceService(name)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
