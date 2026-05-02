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

type SubmitTrainingTaskLogic struct {
	logx.Logger
	ctx       context.Context
	svcCtx    *svc.ServiceContext
	streamKey string
}

func NewSubmitTrainingTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SubmitTrainingTaskLogic {
	return &SubmitTrainingTaskLogic{
		Logger:    logx.WithContext(ctx),
		ctx:       ctx,
		svcCtx:    svcCtx,
		streamKey: svcCtx.Config.Redis.Streams.Training,
	}
}

func (l *SubmitTrainingTaskLogic) SubmitTrainingTask(req *types.SubmitTrainingReq) (resp *types.TrainingTaskResp, err error) {
	// 1. 前置参数校验，拦截无效请求
	if err := l.validateRequest(req); err != nil {
		l.Errorf("submit training task validate failed, %v", err)
		return nil, fmt.Errorf(" validate request failed, %w", err)
	}

	// 2.幂等性控制：基于请求唯一标识防止重复提交
	requestID := req.RequestID
	if requestID == "" {
		requestID = uuid.New().String()
	}
	// 幂等性锁：10分钟内重复请求直接返回
	lockKey := fmt.Sprintf("train:idempotent:%s", requestID)
	ok, err := l.svcCtx.RedisClient.SetNX(l.ctx, lockKey, 1, 10*time.Minute).Result()
	if err != nil {
		l.Errorf("idempotent lock set failed: %v", err)
		return nil, fmt.Errorf("system error, please try again later")
	}
	if !ok {
		l.Infof("duplicate submit request, requestID: %s", requestID)
		return nil, errors.New("duplicate task submission, please do not submit repeatedly")
	}
	// 3. 环境变量转换
	var env []model.EnvVar
	if req.Env != nil {
		for _, v := range req.Env {
			env = append(env, model.EnvVar{
				Name:  v.Name,
				Value: v.Value,
			})
		}
	}
	// 4. 任务初始化，状态流转规范：Submitted -> Queued -> Pending -> Running -> Succeeded/Failed
	maxRetries := l.svcCtx.Config.Redis.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	task := &model.TrainingTask{
		TaskID:        "train-" + uuid.New().String(),
		RequestID:     requestID,
		Name:          req.Name,
		ModelName:     req.ModelName,
		Framework:     req.Framework,
		Image:         req.Image,
		Command:       req.Command,
		Args:          req.Args,
		Resources:     model.ResourceRequest(req.Resources),
		Distributed:   req.Distributed,
		WorkerNum:     req.WorkerNum,
		MasterNum:     req.MasterNum,
		Env:           env,
		DatasetPath:   req.DatasetPath,
		OutputPath:    req.OutputPath,
		Status:        model.StatusSubmitted, // 初始状态：已提交
		Priority:      req.Priority,
		MaxRetries:    maxRetries,
		RetryCount:    0, // 初始化重试次数
		EnableMonitor: req.EnableMonitor,
		EnableLogs:    req.EnableLogs,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// 5. 先持久化到DB，再入队，保证数据一致性
	// 带超时控制的DB操作
	ctx, cancelFunc := context.WithTimeout(l.ctx, 10*time.Second)
	defer cancelFunc()
	if err := l.svcCtx.TrainingTaskRepo.Create(l.ctx, task); err != nil {
		l.Errorf("save training task failed, %v", err)
		// 释放幂等性锁
		l.svcCtx.RedisClient.Del(l.ctx, lockKey)
		return nil, fmt.Errorf("save training task failed, %v", err)
	}
	//6. 任务序列化
	data, err := task.Marshal()
	if err != nil {
		l.Errorf("task marshal failed: %v", err)
		// 回滚DB记录
		_ = l.svcCtx.TrainingTaskRepo.Delete(ctx, task.TaskID)
		_ = l.svcCtx.RedisClient.Del(l.ctx, lockKey).Err()
		return nil, fmt.Errorf("task marshal failed, %v", err)
	}

	// 7. 写入Redis任务队列，带超时控制
	if err := l.svcCtx.TrainingQueue.Push(ctx, task.TaskID, data, task.Priority); err != nil {
		l.Errorf("push task to queue failed: %v", err)
		// 回滚DB记录
		_ = l.svcCtx.TrainingTaskRepo.Delete(ctx, task.TaskID)
		_ = l.svcCtx.RedisClient.Del(l.ctx, lockKey).Err()
		return nil, fmt.Errorf("submit task to queue failed, %w", err)
	}

	// 8. 更新任务状态为Queued（已入队）
	task.Status = model.StatusQueued
	task.UpdatedAt = time.Now()
	if err := l.svcCtx.TrainingTaskRepo.Update(ctx, task); err != nil {
		l.Errorf("update task status to queued failed: %v", err)
		// 不返回错误，任务已入队，不影响核心流程
	}

	l.Infof("training task submitted successfully, taskID: %s", task.TaskID)

	return &types.TrainingTaskResp{
		TaskID:  task.TaskID,
		Status:  string(model.StatusPending),
		Message: "training task submitted successfully",
	}, nil
}

// validateRequest 入参校验
func (l *SubmitTrainingTaskLogic) validateRequest(req *types.SubmitTrainingReq) error {
	if req.Name == "" {
		return errors.New("task name is required")
	}
	if req.ModelName == "" {
		return errors.New("model name is required")
	}
	if req.Framework == "" {
		return errors.New("framework is required, support pytorch/tensorflow/onnx")
	}
	if req.Image == "" {
		return errors.New("image is required")
	}
	if req.Resources.CPU == "" || req.Resources.Memory == "" {
		return errors.New("cpu and memory resources are required")
	}
	if req.Distributed && req.WorkerNum <= 0 {
		return errors.New("workerNum must be greater than 0 for distributed training")
	}
	if req.OutputPath == "" {
		return errors.New("output path is required for model saving")
	}
	return nil
}
