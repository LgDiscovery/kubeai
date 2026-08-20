package task_log

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"kubeai-inference-gateway/internal/svc"
	"kubeai-inference-gateway/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GetTaskLogsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetTaskLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTaskLogsLogic {
	return &GetTaskLogsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetTaskLogs 获取任务 Pod 日志
func (l *GetTaskLogsLogic) GetTaskLogs(req *types.GetTaskLogsReq) (resp *types.TaskLogsResp, err error) {
	namespace := l.svcCtx.Config.K8s.Namespace

	// 查找任务相关的 Pod
	// 训练任务 Pod 命名通常包含 task-id
	podList, err := l.svcCtx.K8sClient.CoreV1().Pods(namespace).List(l.ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("kubeai.io/task-id=%s", req.TaskID),
	})
	if err != nil {
		// 如果按 label 查找失败，尝试按名称模糊匹配
		podList, err = l.svcCtx.K8sClient.CoreV1().Pods(namespace).List(l.ctx, metav1.ListOptions{})
		if err != nil {
			l.Logger.Errorf("List pods failed: %v", err)
			return nil, fmt.Errorf("获取 Pod 列表失败: %w", err)
		}
	}

	// 过滤出匹配的 Pod
	var targetPod *corev1.Pod
	for i := range podList.Items {
		pod := &podList.Items[i]
		if strings.Contains(pod.Name, req.TaskID) {
			targetPod = pod
			break
		}
	}

	if targetPod == nil {
		// 如果没找到，返回空日志和提示
		resp = &types.TaskLogsResp{
			Code:    0,
			Message: "未找到任务 Pod，可能任务尚未启动或已完成",
			Data: types.TaskLogsData{
				TaskID:    req.TaskID,
				PodName:   "",
				Logs:      []string{},
				Timestamp: time.Now().Format(time.RFC3339),
			},
		}
		return
	}

	// 获取 Pod 日志
	logOptions := &corev1.PodLogOptions{
		Timestamps: true,
	}

	// 如果指定了行数
	if req.TailLines > 0 {
		tailLines := int64(req.TailLines)
		logOptions.TailLines = &tailLines
	}

	// 如果指定了容器
	if req.Container != "" {
		logOptions.Container = req.Container
	}

	// 如果指定了起始时间
	if req.SinceTime != "" {
		sinceTime, err := time.Parse(time.RFC3339, req.SinceTime)
		if err == nil {
			t := metav1.NewTime(sinceTime)
			logOptions.SinceTime = &t
		}
	}

	podLogs, err := l.svcCtx.K8sClient.CoreV1().Pods(namespace).GetLogs(targetPod.Name, logOptions).Stream(l.ctx)
	if err != nil {
		l.Logger.Errorf("Get pod logs failed: %v", err)
		return nil, fmt.Errorf("获取 Pod 日志失败: %w", err)
	}
	defer podLogs.Close()

	logBytes, err := io.ReadAll(podLogs)
	if err != nil {
		l.Logger.Errorf("Read pod logs failed: %v", err)
		return nil, fmt.Errorf("读取 Pod 日志失败: %w", err)
	}

	// 按行分割日志
	logLines := strings.Split(strings.TrimSpace(string(logBytes)), "\n")
	if len(logLines) == 1 && logLines[0] == "" {
		logLines = []string{}
	}

	resp = &types.TaskLogsResp{
		Code:    0,
		Message: "success",
		Data: types.TaskLogsData{
			TaskID:    req.TaskID,
			PodName:   targetPod.Name,
			Logs:      logLines,
			Timestamp: time.Now().Format(time.RFC3339),
		},
	}
	return
}

// ListTaskPods 列出任务相关的 Pod
type ListTaskPodsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListTaskPodsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTaskPodsLogic {
	return &ListTaskPodsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListTaskPodsLogic) ListTaskPods(taskID string) (resp *types.TaskPodsResp, err error) {
	namespace := l.svcCtx.Config.K8s.Namespace

	podList, err := l.svcCtx.K8sClient.CoreV1().Pods(namespace).List(l.ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("kubeai.io/task-id=%s", taskID),
	})
	if err != nil {
		l.Logger.Errorf("List task pods failed: %v", err)
		return nil, fmt.Errorf("获取任务 Pod 列表失败: %w", err)
	}

	pods := make([]types.TaskPodItem, 0, len(podList.Items))
	for _, pod := range podList.Items {
		pods = append(pods, types.TaskPodItem{
			Name:      pod.Name,
			Status:    string(pod.Status.Phase),
			NodeName:  pod.Spec.NodeName,
			CreatedAt: pod.CreationTimestamp.Format(time.RFC3339),
		})
	}

	resp = &types.TaskPodsResp{
		Code:    0,
		Message: "success",
		Data: types.TaskPodsData{
			TaskID: taskID,
			Pods:   pods,
			Total:  len(pods),
		},
	}
	return
}
