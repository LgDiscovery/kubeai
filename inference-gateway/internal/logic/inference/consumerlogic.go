package inference

import (
	"context"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"kubeai-inference-gateway/internal/help"
	"time"

	aiv1 "kubeai-inference-gateway/inferenceservice/api/v1"
	"kubeai-inference-gateway/internal/model"
	"kubeai-inference-gateway/internal/svc"
	tiv1 "kubeai-inference-gateway/trainingjob/api/v1"
)

type ConsumerLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewConsumerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConsumerLogic {
	return &ConsumerLogic{ctx: ctx, svcCtx: svcCtx}
}

// ProcessInferenceTask 处理推理任务：创建 InferenceService CRD
func (l *ConsumerLogic) ProcessInferenceTask(taskID string, data []byte) error {
	task, err := model.UnmarshalInferenceTask(data)
	if err != nil {
		return fmt.Errorf("unmarshal inference task failed, %v", err)
	}
	logx.Infof("Processing inference task %s", task.TaskID)

	// 1. 检查是否已存在 InferenceService
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
		logx.Infof("InferenceService %s already exists", isvcName)
		return nil
	}

	// 2. 资源匹配与节点选择
	resourceReq := help.ConvertToResourceRequest(task.Resources)
	nodeName, err := l.svcCtx.ResourceTracker.FindFitNode(resourceReq, l.svcCtx.PlacementStrategy)
	if err != nil {
		logx.Errorf("find fit node failed, %v", err)
		return fmt.Errorf("find fit node failed, %v", err)
	}
	task.ScheduledNode = nodeName

	// 3构建 InferenceService 对象
	isvc := &aiv1.InferenceService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      isvcName,
			Namespace: l.svcCtx.Config.K8s.Namespace,
		},
		Spec: aiv1.InferenceServiceSpec{
			ModelName:    task.ModelName,
			ModelVersion: task.ModelVersion,
			Replicas:     ptr(int32(1)),
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(task.Resources.CPU),
					corev1.ResourceMemory: resource.MustParse(task.Resources.Memory),
				},
			},
		},
	}
	if task.Resources.GPU != "" {
		isvc.Spec.Resources.Requests["nvidia.com/gpu"] = resource.MustParse(task.Resources.GPU)
	}

	// 使用 controller-runtime client 创建
	if err := l.svcCtx.CtrlClient.Create(l.ctx, isvc); err != nil {
		logx.Errorf("Failed to create InferenceService: %v", err)
		return l.handleInferenceTaskFailure(task, err)
	}
	task.Status = model.StatusRunning
	task.UpdatedAt = time.Now()
	logx.Infof("Created InferenceService %s", isvcName)
	return l.saveInferenceTaskState(task)
}

func (l *ConsumerLogic) handleInferenceTaskFailure(task *model.InferenceTask, err error) error {
	task.RetryCount++
	task.ErrorMessage = err.Error()
	task.UpdatedAt = time.Now()
	if task.RetryCount >= task.MaxRetries {
		task.Status = model.StatusFailed
		// 4. 移入死信队列
		data, _ := task.Marshal()
		if err := l.svcCtx.DeadLetterQueue.Push(l.ctx, task.TaskID, data, task.Priority); err != nil {
			return fmt.Errorf("dead letter queue push failed, %v", err)
		}
	} else {
		task.Status = model.StatusPending
		// 重新入队（延迟）
		time.Sleep(time.Duration(task.RetryCount) * time.Second)
		data, _ := task.Marshal()
		if err := l.svcCtx.InferenceQueue.Push(l.ctx, task.TaskID, data, task.Priority); err != nil {
			return fmt.Errorf("inference task push failed, %v", err)
		}
	}
	// 持久化状态...
	return l.saveInferenceTaskState(task)
}

func (l *ConsumerLogic) saveInferenceTaskState(task *model.InferenceTask) error {
	key := fmt.Sprintf("kubeai:task:inference:%s", task.TaskID)
	data, err := task.Marshal()
	if err != nil {
		return fmt.Errorf("marshal inference task failed, %v", err)
	}
	return l.svcCtx.RedisClient.Set(l.ctx, key, data, 24*time.Hour).Err()
}

// ProcessTrainingTask 处理训练任务：创建 TrainingJob CRD
func (l *ConsumerLogic) ProcessTrainingTask(taskID string, data []byte) error {
	task, err := model.UnmarshalTrainingTask(data)
	if err != nil {
		return fmt.Errorf("unmarshal training task failed, %v", err)
	}
	logx.Infof("Processing training task %s", task.TaskID)

	jobName := fmt.Sprintf("train-%s", task.TaskID)

	// 检查是否已存在 TrainingJob
	gvr := schema.GroupVersionResource{
		Group:    tiv1.GroupVersion.Group,
		Version:  tiv1.GroupVersion.Version,
		Resource: "trainingJobs",
	}
	_, err = l.svcCtx.DynamicClient.Resource(gvr).Namespace(l.svcCtx.Config.K8s.Namespace).
		Get(l.ctx, jobName, metav1.GetOptions{})
	if err == nil {
		// 已存在，无需创建
		logx.Infof("TrainingJob %s already exists", jobName)
		return nil
	}

	// 2. 资源匹配与节点选择
	resourceReq := help.ConvertToResourceRequest(task.Resources)
	_, err = l.svcCtx.ResourceTracker.FindFitNode(resourceReq, l.svcCtx.PlacementStrategy)
	if err != nil {
		logx.Errorf("find fit node failed, %v", err)
		return fmt.Errorf("find fit node failed, %v", err)
	}

	// 构建 TrainingJob
	trainingJob := &tiv1.TrainingJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: l.svcCtx.Config.K8s.Namespace,
		},
		Spec: tiv1.TrainingJobSpec{
			Framework:   task.Framework,
			Image:       task.Image,
			Command:     task.Command,
			Distributed: task.Distributed,
			WorkerNum:   task.WorkerNum,
			Resources: tiv1.ResourceRequirements{
				CPU:    task.Resources.CPU,
				Memory: task.Resources.Memory,
				GPU:    task.Resources.GPU,
			},
			Env:          convertEnvVars(task.Env),
			BackoffLimit: 3,
		},
	}

	if err := l.svcCtx.CtrlClient.Create(l.ctx, trainingJob); err != nil {
		logx.Errorf("Failed to create TrainingJob: %v", err)
		return l.handleTrainingTaskFailure(task, err)
	}
	task.Status = model.StatusRunning
	task.UpdatedAt = time.Now()
	logx.Infof("Created TrainingJob %s", jobName)
	return l.saveTrainingTaskState(task)
}

func (l *ConsumerLogic) handleTrainingTaskFailure(task *model.TrainingTask, err error) error {
	task.RetryCount++
	task.ErrorMessage = err.Error()
	task.UpdatedAt = time.Now()
	if task.RetryCount >= task.MaxRetries {
		task.Status = model.StatusFailed
		// 4. 移入死信队列
		data, _ := task.Marshal()
		if err := l.svcCtx.DeadLetterQueue.Push(l.ctx, task.TaskID, data, task.Priority); err != nil {
			return fmt.Errorf("dead letter queue push failed, %v", err)
		}
	} else {
		task.Status = model.StatusPending
		// 重新入队（延迟）
		time.Sleep(time.Duration(task.RetryCount) * time.Second)
		data, _ := task.Marshal()
		if err := l.svcCtx.TrainingQueue.Push(l.ctx, task.TaskID, data, task.Priority); err != nil {
			return fmt.Errorf("training task push failed, %v", err)
		}
	}
	return l.saveTrainingTaskState(task)
}

func (l *ConsumerLogic) saveTrainingTaskState(task *model.TrainingTask) error {
	key := fmt.Sprintf("kubeai:task:training:%s", task.TaskID)
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
