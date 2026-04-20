// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package job_scheduler

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"kubeai-job-scheduler/internal/model"
	"time"

	"kubeai-job-scheduler/internal/svc"
	"kubeai-job-scheduler/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SubmitInferenceTaskLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSubmitInferenceTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SubmitInferenceTaskLogic {
	return &SubmitInferenceTaskLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// SubmitInference 提交推理任务
func (l *SubmitInferenceTaskLogic) SubmitInference(req *types.SubmitInferenceReq) (resp *types.InferenceTaskResp, err error) {
	// 1. 校验模型可用性
	modelVersion, err := l.svcCtx.ModelManagerClient.CheckModelAvailable(l.ctx, req.ModelName, req.ModelVersion)
	if err != nil {
		return nil, fmt.Errorf("model version check failed, %v", err)
	}

	// 2. 获取模型下载URL
	modelURL, err := l.svcCtx.ModelManagerClient.GetModelDownloadURL(l.ctx, req.ModelName, req.ModelVersion)
	if err != nil {
		return nil, fmt.Errorf("model download url get failed, %v", err)
	}

	// 3. 构建任务对象
	task := &model.InferenceTask{
		TaskID:       "inference-" + uuid.New().String()[:8],
		ModelName:    req.ModelName,
		ModelVersion: req.ModelVersion,
		ModelPath:    modelURL, // 直接传递下载 URL
		Framework:    modelVersion.Framework,
		Resources:    model.ResourceRequest(req.Resources),
		InputData:    req.InputData,
		OutputTopic:  req.OutputTopic,
		Status:       model.StatusPending,
		Priority:     req.Priority,
		MaxRetries:   l.svcCtx.Config.Redis.MaxRetries,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// 3. 入队
	data, _ := task.Marshal()
	if err := l.svcCtx.InferenceQueue.Push(l.ctx, task.TaskID, data, task.Priority); err != nil {
		return nil, fmt.Errorf("inference task push failed, %v", err)
	}
	logx.Infof("inference task %s submitted, model: %s/%s", task.TaskID, task.ModelName, task.ModelVersion)

	if err := l.saveInferenceTaskState(task); err != nil {
		return nil, fmt.Errorf("save inference task state failed, %v", err)
	}

	return &types.InferenceTaskResp{
		TaskID:  task.TaskID,
		Status:  string(model.StatusPending),
		Message: "task submitted successfully",
	}, nil
}

func (l *SubmitInferenceTaskLogic) saveInferenceTaskState(task *model.InferenceTask) error {
	key := fmt.Sprintf("kubeai:task:inference:%s", task.TaskID)
	data, err := task.Marshal()
	if err != nil {
		return fmt.Errorf("marshal inference task failed, %v", err)
	}
	return l.svcCtx.RedisClient.Set(l.ctx, key, data, 24*time.Hour).Err()
}
