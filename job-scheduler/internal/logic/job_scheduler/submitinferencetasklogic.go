// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package job_scheduler

import (
	"context"
	"errors"
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
	ctx       context.Context
	svcCtx    *svc.ServiceContext
	streamKey string
}

func NewSubmitInferenceTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SubmitInferenceTaskLogic {
	return &SubmitInferenceTaskLogic{
		Logger:    logx.WithContext(ctx),
		ctx:       ctx,
		svcCtx:    svcCtx,
		streamKey: svcCtx.Config.Redis.Streams.Inference,
	}
}

// SubmitInference 提交推理任务
func (l *SubmitInferenceTaskLogic) SubmitInference(req *types.SubmitInferenceReq) (resp *types.InferenceTaskResp, err error) {
	// 1. 校验请求参数
	if err := l.validate(req); err != nil {
		return nil, err
	}

	// 2. 幂等防重
	requestID := req.RequestID
	if requestID == "" {
		requestID = uuid.New().String()
	}
	lockKey := fmt.Sprintf("inf:idempotent:%s", requestID)
	ok, err := l.svcCtx.RedisClient.SetNX(l.ctx, lockKey, 1, 10*time.Minute).Result()
	if err != nil {
		return nil, fmt.Errorf("idempotent check failed: %w", err)
	}
	if !ok {
		return nil, errors.New("duplicate submit, please retry later")
	}
	// 统一释放锁的defer，避免异常场景锁泄漏
	defer func() {
		_, delErr := l.svcCtx.RedisClient.Del(l.ctx, lockKey).Result()
		if delErr != nil {
			l.Errorf("release idempotent lock failed: %v", delErr)
		}
	}()

	// 3. 校验模型可用性
	modelVersion, err := l.svcCtx.ModelManagerClient.CheckModelAvailable(l.ctx, req.ModelName, req.ModelVersion)
	if err != nil {
		return nil, fmt.Errorf("model unavailable: %w", err)
	}

	// 4. 获取模型下载URL
	modelURL, err := l.svcCtx.ModelManagerClient.GetModelDownloadURL(l.ctx, req.ModelName, req.ModelVersion)
	if err != nil {
		return nil, fmt.Errorf("model download url get failed, %v", err)
	}

	// 5. 构建任务对象
	maxRetries := l.svcCtx.Config.Redis.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	task := &model.InferenceTask{
		TaskID:       "inference-" + uuid.New().String(),
		RequestID:    requestID,
		ModelName:    req.ModelName,
		ModelVersion: req.ModelVersion,
		ModelPath:    modelURL, // 直接传递下载 URL
		Framework:    modelVersion.Framework,
		Resources:    model.ResourceRequest(req.Resources),
		InputData:    req.InputData,
		OutputTopic:  req.OutputTopic,
		Status:       model.StatusPending,
		Priority:     req.Priority,
		RetryCount:   0,
		MaxRetries:   maxRetries,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	//6. 先落库 一致性核心
	ctx, cancel := context.WithTimeout(l.ctx, 10*time.Second)
	defer cancel()
	if err := l.svcCtx.InferenceTaskRepo.Create(ctx, task); err != nil {
		_ = l.svcCtx.RedisClient.Del(l.ctx, lockKey).Err()
		return nil, fmt.Errorf("save task failed: %w", err)
	}
	// 7. 序列化任务
	data, err := task.Marshal()
	if err != nil {
		// 先删除数据库中的任务
		_ = l.svcCtx.InferenceTaskRepo.Delete(l.ctx, task.TaskID)
		_ = l.svcCtx.RedisClient.Del(l.ctx, lockKey).Err()
		return nil, fmt.Errorf("marshal task failed: %w", err)
	}
	// 8. 入队
	if err := l.svcCtx.InferenceQueue.Push(l.ctx, task.TaskID, data, task.Priority); err != nil {
		// 先删除数据库中的任务
		_ = l.svcCtx.InferenceTaskRepo.Delete(l.ctx, task.TaskID)
		_ = l.svcCtx.RedisClient.Del(l.ctx, lockKey).Err()
		return nil, fmt.Errorf("inference task push failed, %w", err)
	}
	logx.Infof("inference task %s submitted, model: %s/%s", task.TaskID, task.ModelName, task.ModelVersion)

	// 9. 更新为已入队
	task.Status = model.StatusQueued
	task.UpdatedAt = time.Now()
	if err := l.svcCtx.InferenceTaskRepo.Update(ctx, task); err != nil {
		l.Errorf("update task status to queued failed: %v, taskID: %s", err, task.TaskID)
		// 不返回错误，任务已入队，异步重试更新状态
		go func() {
			for i := 0; i < 3; i++ {
				time.Sleep(time.Duration(i+1) * time.Second)
				if err := l.svcCtx.InferenceTaskRepo.Update(context.Background(), task); err == nil {
					l.Infof("retry update task status success, taskID: %s", task.TaskID)
					return
				}
			}
			l.Errorf("all retry update task status failed, taskID: %s", task.TaskID)
		}()
	}
	l.Infof("inference task %s submitted, model: %s/%s", task.TaskID, task.ModelName, task.ModelVersion)

	return &types.InferenceTaskResp{
		TaskID:  task.TaskID,
		Status:  string(model.StatusPending),
		Message: "task submitted successfully",
	}, nil
}

func (l *SubmitInferenceTaskLogic) validate(req *types.SubmitInferenceReq) error {
	if req.ModelName == "" {
		return errors.New("modelName required")
	}
	if req.ModelVersion == "" {
		return errors.New("modelVersion required")
	}
	if req.Resources.CPU == "" || req.Resources.Memory == "" {
		return errors.New("cpu/mem required")
	}
	if req.Priority <= 0 {
		return errors.New("priority must be greater than 0")
	}
	if req.MaxRetries <= 0 {
		return errors.New("maxRetries must be greater than 0")
	}
	if req.OutputTopic == "" {
		return errors.New("outputTopic required")
	}
	return nil
}
