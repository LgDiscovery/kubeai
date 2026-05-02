package inference

import (
	"context"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"kubeai-inference-gateway/internal/help"
	"math"
	"time"

	aiv1 "kubeai-inference-gateway/inferenceservice/api/v1"
	"kubeai-inference-gateway/internal/model"
	"kubeai-inference-gateway/internal/svc"
	tiv1 "kubeai-inference-gateway/trainingjob/api/v1"
)

const (
	// 模型注册最大重试次数
	maxModelRegisterRetry = 10
)

type ConsumerLogic struct {
	ctx       context.Context
	svcCtx    *svc.ServiceContext
	streamKey string
}

func NewConsumerLogic(ctx context.Context, svcCtx *svc.ServiceContext, streamKey string) *ConsumerLogic {
	return &ConsumerLogic{ctx: ctx, svcCtx: svcCtx, streamKey: streamKey}
}

// ProcessInferenceTask 处理推理任务：创建 InferenceService CRD
func (l *ConsumerLogic) ProcessInferenceTask(taskID string, data []byte) error {
	log := logx.WithContext(l.ctx).WithFields(
		logx.Field("taskID", taskID),
		logx.Field("taskType", model.InferenceTaskType),
	)
	// 1. 先反序列化，再处理业务逻辑
	task, err := model.UnmarshalInferenceTask(data)
	if err != nil {
		log.Errorf("unmarshal inference task failed: %v", err)
		// 反序列化失败直接进死信，无重试意义
		return l.moveToDeadLetter(task, err, "inference")
	}

	// 2. 幂等性校验：终态任务直接跳过，ACK消息
	existTask, err := l.svcCtx.InferenceTaskRepo.GetByTaskID(l.ctx, taskID)
	if err != nil {
		log.Errorf("get inference task failed: %v", err)
		return fmt.Errorf("get inference task failed: %w", err)
	}
	if existTask.Status == model.StatusSucceeded || existTask.Status == model.StatusFailed {
		log.Infof("task already in terminal state: %s, skip processing", existTask.Status)
		// 终态任务直接ACK，不重复处理
		return nil
	}

	log.Infof("start processing inference task %s", taskID)
	// 3. 检查是否已存在 InferenceService
	isvcName := fmt.Sprintf("%s-%s", task.ModelName, task.ModelVersion)
	gvr := schema.GroupVersionResource{
		Group:    aiv1.GroupVersion.Group,
		Version:  aiv1.GroupVersion.Version,
		Resource: "inferenceServices",
	}
	_, err = l.svcCtx.DynamicClient.Resource(gvr).Namespace(l.svcCtx.Config.K8s.Namespace).
		Get(l.ctx, isvcName, metav1.GetOptions{})
	if err == nil {
		// 已存在，无需创建
		log.Infof("InferenceService %s already exists, skip creation", isvcName)
		// 更新任务状态为运行中
		task.Status = model.StatusRunning
		task.UpdatedAt = time.Now()
		if err := l.svcCtx.InferenceTaskRepo.Update(l.ctx, task); err != nil {
			log.Errorf("update task status failed: %v", err)
			return err
		}
		_ = l.saveInferenceTaskState(task)
		return nil
	}

	// 仅当确定是NotFound错误时，才走创建逻辑，其他错误直接重试
	if !errors.IsNotFound(err) {
		log.Errorf("check InferenceService exist failed: %v", err)
		return fmt.Errorf("check InferenceService exist failed: %w", err)
	}

	// 4. 资源匹配与节点选择
	resourceReq := help.ConvertToResourceRequest(task.Resources)
	nodeName, err := l.svcCtx.ResourceTracker.FindFitNode(resourceReq, l.svcCtx.PlacementStrategy)
	if err != nil {
		log.Errorf("find fit node failed: %v", err)
		return l.handleInferenceTaskFailure(task, fmt.Errorf("find fit node failed: %w", err))
	}
	task.ScheduledNode = nodeName
	task.PodName = fmt.Sprintf("%s-%s", task.ModelName, task.ModelVersion)

	// 5 构建 InferenceService 对象
	isvc := &aiv1.InferenceService{
		TypeMeta: metav1.TypeMeta{
			Kind:       "InferenceService",
			APIVersion: aiv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      isvcName,
			Namespace: l.svcCtx.Config.K8s.Namespace,
		},
		Spec: aiv1.InferenceServiceSpec{
			ModelName:    task.ModelName,
			ModelVersion: task.ModelVersion,
			Replicas:     ptr(int32(1)),
			NodeName:     nodeName,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(task.Resources.CPU),
					corev1.ResourceMemory: resource.MustParse(task.Resources.Memory),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(task.Resources.CPU),
					corev1.ResourceMemory: resource.MustParse(task.Resources.Memory),
				},
			},
		},
	}
	if task.Resources.GPU != "" && task.Resources.GPU != "0" {
		gpuQuantity := resource.MustParse(task.Resources.GPU)
		isvc.Spec.Resources.Requests["nvidia.com/gpu"] = gpuQuantity
		isvc.Spec.Resources.Limits["nvidia.com/gpu"] = gpuQuantity
	}

	// 6. 创建CRD，先更新DB状态，再创建资源，保证状态一致性
	task.Status = model.StatusPending
	task.UpdatedAt = time.Now()
	if err := l.svcCtx.InferenceTaskRepo.Update(l.ctx, task); err != nil {
		log.Errorf("update task status to pending failed: %v", err)
		// 释放节点预留
		_ = l.svcCtx.ResourceTracker.ReleaseNodeReserve(nodeName, task.TaskID)
		return err

	}
	// 7. 使用 controller-runtime client 创建 InferenceService
	if err := l.svcCtx.CtrlClient.Create(l.ctx, isvc); err != nil {
		logx.Errorf("Failed to create InferenceService: %v", err)
		// 释放节点预留
		_ = l.svcCtx.ResourceTracker.ReleaseNodeReserve(nodeName, task.TaskID)
		return l.handleInferenceTaskFailure(task, err)
	}
	task.Status = model.StatusRunning
	task.UpdatedAt = time.Now()
	log.Infof("Created InferenceService %s successfully", isvcName)
	if err := l.svcCtx.InferenceTaskRepo.Update(l.ctx, task); err != nil {
		log.Errorf("update inference task status failed: %v", err)
		// 不返回错误，CRD已创建，不影响核心流程
	}
	_ = l.saveInferenceTaskState(task)
	// 处理成功后ACK消息，避免重复投递
	return nil
}

// ProcessTrainingTask 处理训练任务：创建 TrainingJob CRD
func (l *ConsumerLogic) ProcessTrainingTask(taskID string, data []byte) error {
	log := logx.WithContext(l.ctx).WithFields(
		logx.Field("taskID", taskID),
		logx.Field("taskType", model.TrainingTaskType))

	// 1. 先反序列化，再处理业务逻辑
	task, err := model.UnmarshalTrainingTask(data)
	if err != nil {
		log.Errorf("unmarshal training task failed: %v", err)
		// 反序列化失败直接进死信
		return l.moveToDeadLetter(task, err, "training")
	}

	// 2. 幂等性校验：终态任务直接跳过，ACK消息
	existTask, err := l.svcCtx.TrainingTaskRepo.GetByTaskID(l.ctx, taskID)
	if err != nil {
		log.Errorf("get training task failed: %v", err)
		return fmt.Errorf("get training task failed: %w", err)
	}
	if existTask.Status == model.StatusSucceeded || existTask.Status == model.StatusFailed {
		log.Infof("task already in terminal state: %s, skip processing", existTask.Status)
		return nil
	}
	log.Infof("start processing training task %s", taskID)
	// 3.校验模型信息
	md, err := l.svcCtx.ModelMgrClient.GetModel(l.ctx, task.ModelName)
	if err != nil {
		log.Errorf("get model failed, %v", err)
		return l.handleTrainingTaskFailure(task, fmt.Errorf("get model failed: %w", err))
	}

	// 4. 检查CRD是否已存在
	jobName := taskID
	gvr := schema.GroupVersionResource{
		Group:    tiv1.GroupVersion.Group,
		Version:  tiv1.GroupVersion.Version,
		Resource: "trainingJobs",
	}
	_, err = l.svcCtx.DynamicClient.Resource(gvr).Namespace(l.svcCtx.Config.K8s.Namespace).
		Get(l.ctx, jobName, metav1.GetOptions{})
	if err == nil {
		log.Infof("TrainingJob %s already exists, skip creation", jobName)
		// 更新任务状态为Running
		task.Status = model.StatusRunning
		task.PodName = jobName
		task.UpdatedAt = time.Now()
		if err := l.svcCtx.TrainingTaskRepo.Update(l.ctx, task); err != nil {
			log.Errorf("update task status failed: %v", err)
			return err
		}
		_ = l.saveTrainingTaskState(task)
		return nil
	}

	// 仅NotFound错误才走创建逻辑
	if !errors.IsNotFound(err) {
		log.Errorf("check TrainingJob exist failed: %v", err)
		return fmt.Errorf("check TrainingJob exist failed: %w", err)
	}

	// 5. 资源匹配与节点选择+预留
	resourceReq := help.ConvertToResourceRequest(task.Resources)
	nodeName, err := l.svcCtx.ResourceTracker.FindFitNodeAndReserve(resourceReq, l.svcCtx.PlacementStrategy, taskID)
	if err != nil {
		logx.Errorf("find fit node failed, %v", err)
		return l.handleTrainingTaskFailure(task, fmt.Errorf("find fit node failed: %w", err))
	}
	task.ScheduledNode = nodeName
	task.PodName = jobName

	// 6. 补全MinIO数据集/输出路径挂载
	volumes, volumeMounts := l.buildMinIOVolume(task.DatasetPath, task.OutputPath)
	if task.Volumes == nil {
		task.Volumes = make([]model.Volume, 0)
	}
	if task.VolumeMounts == nil {
		task.VolumeMounts = make([]model.VolumeMount, 0)
	}
	task.Volumes = append(task.Volumes, volumes...)
	task.VolumeMounts = append(task.VolumeMounts, volumeMounts...)

	// 7. 构建 TrainingJob CRD，【修复】补全所有核心参数
	trainingJob := &tiv1.TrainingJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TrainingJob",
			APIVersion: tiv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: l.svcCtx.Config.K8s.Namespace,
		},
		Spec: tiv1.TrainingJobSpec{
			Framework:   task.Framework,
			Image:       task.Image,
			Args:        task.Args,
			Command:     task.Command,
			Distributed: task.Distributed,
			WorkerNum:   task.WorkerNum,
			MasterNum:   task.MasterNum,
			NodeName:    nodeName, // 写入筛选节点，精准调度
			Resources: tiv1.ResourceRequirements{
				CPU:    task.Resources.CPU,
				Memory: task.Resources.Memory,
				GPU:    task.Resources.GPU,
			},
			Env:                     convertEnvVars(task.Env),
			Volumes:                 convertVolumes(task.Volumes),
			VolumeMounts:            convertVolumeMounts(task.VolumeMounts),
			DatasetPath:             task.DatasetPath, // 传入数据集路径
			OutputPath:              task.OutputPath,  // 传入输出路径
			BackoffLimit:            int32(task.MaxRetries),
			TTLSecondsAfterFinished: 3600,
			ActiveDeadlineSeconds:   86400,
			ModelID:                 string(md.ID),
			ModelName:               md.Name,
			EnableMonitor:           task.EnableMonitor,
			EnableLogs:              task.EnableLogs,
		},
	}

	// 8. 先更新DB状态为Pending，再创建CRD
	task.Status = model.StatusPending
	task.UpdatedAt = time.Now()
	if err := l.svcCtx.TrainingTaskRepo.Update(l.ctx, task); err != nil {
		log.Errorf("update task status to pending failed: %v", err)
		_ = l.svcCtx.ResourceTracker.ReleaseNodeReserve(nodeName, task.TaskID)
		return err
	}

	// 9. 创建CRD
	if err := l.svcCtx.CtrlClient.Create(l.ctx, trainingJob); err != nil {
		log.Errorf("Failed to create TrainingJob: %v", err)
		_ = l.svcCtx.ResourceTracker.ReleaseNodeReserve(nodeName, task.TaskID)
		return l.handleTrainingTaskFailure(task, err)
	}

	// 10. 创建成功，更新状态为Running
	task.Status = model.StatusRunning
	task.UpdatedAt = time.Now()
	log.Infof("Created TrainingJob %s successfully", jobName)
	if err := l.svcCtx.TrainingTaskRepo.Update(l.ctx, task); err != nil {
		log.Errorf("update training task status failed: %v", err)
	}

	_ = l.saveTrainingTaskState(task)
	return nil
}

func (l *ConsumerLogic) handleInferenceTaskFailure(task *model.InferenceTask, err error) error {
	log := logx.WithContext(l.ctx).WithFields(logx.Field("taskID", task.TaskID))
	task.RetryCount++
	task.ErrorMessage = err.Error()
	task.UpdatedAt = time.Now()
	// 释放节点预留
	if task.ScheduledNode != "" {
		_ = l.svcCtx.ResourceTracker.ReleaseNodeReserve(task.ScheduledNode, task.TaskID)
	}

	// 超过最大重试次数，移入死信队列
	if task.RetryCount >= task.MaxRetries {
		log.Errorf("task retry count exceed max retries %d, move to dead letter", task.MaxRetries)
		task.Status = model.StatusFailed
		if err := l.svcCtx.InferenceTaskRepo.Update(l.ctx, task); err != nil {
			log.Errorf("update task status to failed failed: %v", err)
			return err
		}
		if err := l.moveToDeadLetter(task, err, "inference"); err != nil {
			log.Errorf("move to dead letter failed: %v", err)
			return err
		}
		_ = l.saveInferenceTaskState(task)
		// 终态任务ACK
		return nil
	}

	// 未超过重试次数，使用指数退避延迟入队，【修复】移除sleep阻塞
	task.Status = model.StatusQueued
	delaySeconds := int64(math.Pow(2, float64(task.RetryCount)))
	if delaySeconds > 60 {
		delaySeconds = 60 // 最大延迟60秒
	}
	data, err := task.Marshal()
	if err != nil {
		log.Errorf("task marshal failed: %v", err)
		return err
	}

	// 写入延迟队列，使用Redis ZSet实现，score为执行时间戳
	execTime := time.Now().Unix() + delaySeconds
	if err := l.svcCtx.RedisClient.ZAdd(l.ctx, l.svcCtx.Config.Redis.Streams.DeadLetter, redis.Z{Score: float64(execTime), Member: data}).Err(); err != nil {
		log.Errorf("push to dead letter queue failed: %v", err)
		return err
	}

	// 更新DB状态
	if err := l.svcCtx.InferenceTaskRepo.Update(l.ctx, task); err != nil {
		log.Errorf("update task status failed: %v", err)
		return err
	}
	_ = l.saveInferenceTaskState(task)
	// 重试任务ACK，避免重复投递
	return nil
}

func (l *ConsumerLogic) saveInferenceTaskState(task *model.InferenceTask) error {
	key := fmt.Sprintf("%s:%s", l.streamKey, task.TaskID)
	data, err := task.Marshal()
	if err != nil {
		return fmt.Errorf("marshal inference task failed, %v", err)
	}
	return l.svcCtx.RedisClient.Set(l.ctx, key, data, 24*time.Hour).Err()
}

func (l *ConsumerLogic) handleTrainingTaskFailure(task *model.TrainingTask, err error) error {
	log := logx.WithContext(l.ctx).WithFields(logx.Field("taskID", task.TaskID))
	task.RetryCount++
	task.ErrorMessage = err.Error()
	task.UpdatedAt = time.Now()

	// 释放节点预留
	if task.ScheduledNode != "" {
		_ = l.svcCtx.ResourceTracker.ReleaseNodeReserve(task.ScheduledNode, task.TaskID)
	}

	if task.RetryCount >= task.MaxRetries {
		log.Errorf("task retry count exceed max retries %d, move to dead letter", task.MaxRetries)
		task.Status = model.StatusFailed
		if err := l.svcCtx.TrainingTaskRepo.Update(l.ctx, task); err != nil {
			log.Errorf("update task status to failed failed: %v", err)
			return err
		}
		if err := l.moveToDeadLetter(task, err, "training"); err != nil {
			log.Errorf("move to dead letter failed: %v", err)
			return err
		}
		_ = l.saveTrainingTaskState(task)
		return nil
	}
	task.Status = model.StatusQueued
	delaySeconds := int64(math.Pow(2, float64(task.RetryCount)))
	if delaySeconds > 60 {
		delaySeconds = 60
	}
	data, err := task.Marshal()
	if err != nil {
		log.Errorf("task marshal failed: %v", err)
		return err
	}

	execTime := time.Now().Unix() + delaySeconds
	if err := l.svcCtx.RedisClient.ZAdd(l.ctx, l.svcCtx.Config.Redis.Streams.DeadLetter, redis.Z{Score: float64(execTime), Member: data}).Err(); err != nil {
		log.Errorf("push to delay queue failed: %v", err)
		return err
	}

	if err := l.svcCtx.TrainingTaskRepo.Update(l.ctx, task); err != nil {
		log.Errorf("update task status failed: %v", err)
		return err
	}
	_ = l.saveTrainingTaskState(task)
	return nil
}

// moveToDeadLetter 移入死信队列，失败不阻塞主流程
func (l *ConsumerLogic) moveToDeadLetter(task interface{}, err error, taskType string) error {
	var data []byte
	var taskID string
	switch t := task.(type) {
	case *model.TrainingTask:
		data, _ = t.Marshal()
		taskID = t.TaskID
	case *model.InferenceTask:
		data, _ = t.Marshal()
		taskID = t.TaskID
	default:
		return fmt.Errorf("invalid task type: %T", t)
	}

	pushErr := l.svcCtx.DeadLetterQueue.Push(l.ctx, taskID, data, err.Error(), taskType)
	if pushErr != nil {
		logx.Errorf("dead letter queue push failed, taskID: %s, err: %v", taskID, pushErr)
		// 死信写入失败，记录日志，不返回错误，避免消息无限重试
		return nil
	}
	return nil
}

// buildMinIOVolume 构建MinIO数据集/输出路径挂载
func (l *ConsumerLogic) buildMinIOVolume(datasetPath, outputPath string) ([]model.Volume, []model.VolumeMount) {
	volumes := make([]model.Volume, 0)
	volumeMounts := make([]model.VolumeMount, 0)

	// 数据集PVC挂载
	if datasetPath != "" {
		volumes = append(volumes, model.Volume{
			Name:      "dataset",
			ClaimName: l.svcCtx.Config.MinIO.DatasetPvc,
		})
		volumeMounts = append(volumeMounts, model.VolumeMount{
			Name:      "dataset",
			MountPath: "/dataset",
		})
	}

	// 模型输出PVC挂载
	if outputPath != "" {
		volumes = append(volumes, model.Volume{
			Name:      "model-output",
			ClaimName: l.svcCtx.Config.MinIO.OutputPvc,
		})
		volumeMounts = append(volumeMounts, model.VolumeMount{
			Name:      "model-output",
			MountPath: "/model-output",
		})
	}

	return volumes, volumeMounts
}

func (l *ConsumerLogic) saveTrainingTaskState(task *model.TrainingTask) error {
	key := fmt.Sprintf("%s:%s", l.streamKey, task.TaskID)
	data, err := task.Marshal()
	if err != nil {
		return fmt.Errorf("marshal training task failed, %v", err)
	}
	return l.svcCtx.RedisClient.Set(l.ctx, key, data, 24*time.Hour).Err()
}

func ptr[T any](v T) *T { return &v }

func convertEnvVars(envs []model.EnvVar) []corev1.EnvVar {
	result := make([]corev1.EnvVar, len(envs))
	for i, e := range envs {
		result[i] = corev1.EnvVar{Name: e.Name, Value: e.Value}
	}
	return result
}

func convertVolumes(vols []model.Volume) []corev1.Volume {
	result := make([]corev1.Volume, len(vols))
	for i, v := range vols {
		result[i] = corev1.Volume{
			Name: v.Name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: v.ClaimName,
				},
			},
		}
	}
	return result
}

func convertVolumeMounts(mounts []model.VolumeMount) []corev1.VolumeMount {
	result := make([]corev1.VolumeMount, len(mounts))
	for i, m := range mounts {
		result[i] = corev1.VolumeMount{
			Name:      m.Name,
			MountPath: m.MountPath,
		}
	}
	return result
}
