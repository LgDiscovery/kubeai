package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

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
	var task model.InferenceTask
	if err := json.Unmarshal(data, &task); err != nil {
		return err
	}
	logx.Infof("Processing inference task %s", task.TaskID)

	// 检查是否已存在 InferenceService
	isvcName := fmt.Sprintf("%s-%s", task.ModelName, task.ModelVersion)
	gvr := schema.GroupVersionResource{
		Group:    aiv1.GroupVersion.Group,
		Version:  aiv1.GroupVersion.Version,
		Resource: "inferenceservices",
	}
	_, err := l.svcCtx.DynamicClient.Resource(gvr).Namespace(l.svcCtx.Config.K8s.Namespace).
		Get(l.ctx, isvcName, metav1.GetOptions{})
	if err == nil {
		// 已存在，无需创建
		logx.Infof("InferenceService %s already exists", isvcName)
		return nil
	}

	// 构建 InferenceService 对象
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
		return err
	}
	logx.Infof("Created InferenceService %s", isvcName)
	return nil
}

// ProcessTrainingTask 处理训练任务：创建 TrainingJob CRD
func (l *ConsumerLogic) ProcessTrainingTask(taskID string, data []byte) error {
	var task model.TrainingTask
	if err := json.Unmarshal(data, &task); err != nil {
		return err
	}
	logx.Infof("Processing training task %s", task.TaskID)

	jobName := fmt.Sprintf("train-%s", task.TaskID)
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
		return err
	}
	logx.Infof("Created TrainingJob %s", jobName)
	return nil
}

func ptr[T any](v T) *T { return &v }

func convertEnvVars(envs []model.EnvVar) []corev1.EnvVar {
	result := make([]corev1.EnvVar, len(envs))
	for i, e := range envs {
		result[i] = corev1.EnvVar{Name: e.Name, Value: e.Value}
	}
	return result
}
