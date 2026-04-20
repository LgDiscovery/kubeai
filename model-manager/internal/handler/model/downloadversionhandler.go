// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package model

import (
	"io"
	"kubeai-model-manager/internal/types"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"kubeai-model-manager/internal/logic/model"
	"kubeai-model-manager/internal/svc"
)

func DownloadVersionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := model.NewDownloadVersionLogic(r.Context(), svcCtx)
		var req types.CommonReq
		if err := httpx.ParsePath(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		reader, storagePath, err := l.DownloadVersion(req.Name, req.Version)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		defer reader.Close()

		// 设置下载响应头
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment; filename="+storagePath)
		// 写入响应体
		if _, err := io.Copy(w, reader); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		if req.Presigned {
			httpx.OkJsonCtx(r.Context(), w, &types.DownloadResp{
				StoragePath: storagePath,
			})
		} else {
			httpx.OkJsonCtx(r.Context(), w, &types.DownloadResp{})
		}
	}
}
