// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package job_scheduler

import (
	"kubeai-job-scheduler/internal/types"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"kubeai-job-scheduler/internal/logic/job_scheduler"
	"kubeai-job-scheduler/internal/svc"
)

func RetryTaskHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.TaskControlReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		l := job_scheduler.NewRetryTaskLogic(r.Context(), svcCtx)
		resp, err := l.RetryTask(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
