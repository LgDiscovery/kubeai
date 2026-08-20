package task_log

import (
	"errors"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"kubeai-inference-gateway/internal/logic/task_log"
	"kubeai-inference-gateway/internal/svc"
	"kubeai-inference-gateway/internal/types"
)

// GetTaskLogsHandler 获取任务日志
func GetTaskLogsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetTaskLogsReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		// 从路径参数获取 task_id
		taskID := r.PathValue("task_id")
		if taskID != "" {
			req.TaskID = taskID
		}
		if req.TaskID == "" {
			httpx.ErrorCtx(r.Context(), w, errors.New("task_id is required"))
			return
		}
		l := task_log.NewGetTaskLogsLogic(r.Context(), svcCtx)
		resp, err := l.GetTaskLogs(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

// ListTaskPodsHandler 列出任务相关 Pod
func ListTaskPodsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		taskID := r.PathValue("task_id")
		if taskID == "" {
			httpx.ErrorCtx(r.Context(), w, errors.New("task_id is required"))
			return
		}
		l := task_log.NewListTaskPodsLogic(r.Context(), svcCtx)
		resp, err := l.ListTaskPods(taskID)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
