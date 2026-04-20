// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package job_scheduler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"kubeai-job-scheduler/internal/logic/job_scheduler"
	"kubeai-job-scheduler/internal/svc"
	"kubeai-job-scheduler/internal/types"
)

func ListTasksHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ListTasksReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := job_scheduler.NewListTasksLogic(r.Context(), svcCtx)
		resp, err := l.ListTasks(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
