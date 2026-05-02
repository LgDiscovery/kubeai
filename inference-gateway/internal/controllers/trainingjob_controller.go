/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	callbackClient "kubeai-inference-gateway/internal/client"
	"kubeai-inference-gateway/internal/help"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	// Kubeflow PyTorchJob 依赖
	trainingv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	aiv1 "kubeai-inference-gateway/trainingjob/api/v1"
)

// 定义 ConditionType 常量（遵循 Kubernetes 惯例）
const (
	ConditionTypeJobReady        = "JobReady"        // 任务是否准备好运行
	ConditionTypeJobSucceeded    = "JobSucceeded"    // 任务是否成功
	ConditionTypeJobFailed       = "JobFailed"       // 任务是否失败
	ConditionTypeModelRegistered = "ModelRegistered" // 模型是否注册成功
	// TrainingJobFinalizer Finalizer 名称
	TrainingJobFinalizer = "training.kubeai.platform.io/finalizer"
	// 模型注册最大重试次数
	maxModelRegisterRetry = 10
)

// TrainingJobReconciler reconciles a TrainingJob object
type TrainingJobReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	ModelManagerAddr  string                             // 模型服务地址
	JobScheduleAddr   string                             // 任务调度服务地址
	ModelClient       *callbackClient.ModelManagerClient //客户端实例
	JobScheduleClient *callbackClient.JobScheduleClient  //客户端实例
}

// +kubebuilder:rbac:groups=training.kubeai.platform.io,resources=trainingjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=training.kubeai.platform.io,resources=trainingjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=training.kubeai.platform.io,resources=trainingjobs/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=kubeflow.org,resources=pytorchjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubeflow.org,resources=pytorchjobs/status,verbs=get
// +kubebuilder:rbac:groups=kubeflow.org,resources=tfjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubeflow.org,resources=tfjobs/status,verbs=get
func (r *TrainingJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("TrainingJob", req.NamespacedName)

	// 1.获取 TrainingJob CR
	var tj aiv1.TrainingJob
	if err := r.Get(ctx, req.NamespacedName, &tj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	originalTJ := tj.DeepCopy()

	// 2.处理Finalizer 生命周期管理
	if tj.ObjectMeta.DeletionTimestamp.IsZero() {
		// 资源未被删除：添加 Finalizer
		if !help.ContainsString(tj.ObjectMeta.Finalizers, TrainingJobFinalizer) {
			tj.ObjectMeta.Finalizers = append(tj.ObjectMeta.Finalizers, TrainingJobFinalizer)
			if err := r.Update(ctx, &tj); err != nil {
				log.Error(err, "Failed to add finalizer to TrainingJob")
				return ctrl.Result{}, err
			}
			// Finalizer更新后，重新入队
			return ctrl.Result{Requeue: true}, nil
		}
	} else {
		// 资源被删除：执行清理逻辑
		if help.ContainsString(tj.ObjectMeta.Finalizers, TrainingJobFinalizer) {
			// 清理资源预留、回调业务侧任务删除
			log.Info("start cleaning up TrainingJob resources")
			tj.ObjectMeta.Finalizers = help.RemoveString(tj.ObjectMeta.Finalizers, TrainingJobFinalizer)
			if err := r.Update(ctx, &tj); err != nil {
				log.Error(err, "Failed to delete finalizer from TrainingJob")
				return ctrl.Result{}, err
			}
		}
		// 已删除完成，无需继续调和
		return ctrl.Result{}, nil
	}

	// 3. 终端状态直接退出（Succeeded/Failed），但模型未注册时仍需尝试注册
	if tj.Status.Phase == "Succeeded" || tj.Status.Phase == "Failed" {
		// 训练成功但模型未注册，且未超过最大重试次数
		if tj.Status.Phase == "Succeeded" && !r.isModelRegistered(&tj) && tj.Status.ModelRegisterRetryCount < maxModelRegisterRetry {
			log.Info("start register trained model to model manager")
			if err := r.registerTrainedModel(ctx, &tj); err != nil {
				log.Error(err, "Failed to register model after job succeeded")
				tj.Status.ModelRegisterRetryCount++
				r.setCondition(&tj, ConditionTypeModelRegistered, metav1.ConditionFalse, "ModelRegisterFailed", fmt.Sprintf("Model register failed: %s", err), metav1.Now())
				// 状态Patch
				if err := r.Status().Patch(ctx, &tj, client.MergeFrom(originalTJ)); err != nil {
					log.Error(err, "Failed to update model register retry status")
					return ctrl.Result{}, err
				}
				// 指数退避重试
				requeueAfter := time.Duration(tj.Status.ModelRegisterRetryCount*10) * time.Second
				return ctrl.Result{RequeueAfter: requeueAfter}, nil
			}
			// 注册成功更新状态
			r.setCondition(&tj, ConditionTypeModelRegistered, metav1.ConditionTrue, "ModelRegistered", "Model registered successfully", metav1.Now())
			log.Info("model registered successfully", "modelID", tj.Status.RegisteredModelID)
			// 回调业务侧任务完成
			go r.callbackTaskStatus(&tj)
			// 统一Patch状态
			if err := r.Status().Patch(ctx, &tj, client.MergeFrom(originalTJ)); err != nil {
				log.Error(err, "Failed to update model registered status")
				return ctrl.Result{}, err
			}
		}
		// 终态直接退出，不再调和
		return ctrl.Result{}, nil
	}

	// 4. 任务超时检查
	if tj.Spec.ActiveDeadlineSeconds > 0 {
		runTime := time.Since(tj.CreationTimestamp.Time).Seconds()
		if runTime > float64(tj.Spec.ActiveDeadlineSeconds) {
			log.Error(nil, "TrainingJob exceeded active deadline", "runTime", runTime, "deadline", tj.Spec.ActiveDeadlineSeconds)
			r.updateStatus(ctx, &tj, "Failed", "JobTimeout", fmt.Sprintf("Job exceeded active deadline of %d seconds", tj.Spec.ActiveDeadlineSeconds))
			_ = r.Status().Patch(ctx, &tj, client.MergeFrom(originalTJ))
			go r.callbackTaskStatus(&tj)
			return ctrl.Result{}, nil
		}
	}

	// 5. 根据 Framework 构建对应Job
	var job client.Object
	var err error
	switch tj.Spec.Framework {
	case "pytorch":
		if tj.Spec.Distributed {
			job, err = r.buildDistributedPyTorchJob(&tj)
		} else {
			job, err = r.buildSingleNodeJob(&tj, "pytorch-single")
		}
	case "tensorflow":
		if tj.Spec.Distributed {
			job, err = r.buildDistributedTFJob(&tj)
		} else {
			job, err = r.buildSingleNodeJob(&tj, "tensorflow-single")
		}
	case "onnx":
		job, err = r.buildSingleNodeJob(&tj, "onnx-single")
	default:
		err = fmt.Errorf("unsupported framework: %s", tj.Spec.Framework)
	}
	if err != nil {
		log.Error(err, "Failed to build job")
		r.updateStatus(ctx, &tj, "Failed", "JobBuildFailed", err.Error())
		return ctrl.Result{}, err
	}

	// 6. 设置 OwnerReference（级联删除）
	if err := ctrl.SetControllerReference(&tj, job, r.Scheme); err != nil {
		log.Error(err, "Failed to set controller reference")
		r.updateStatus(ctx, &tj, "Failed", "OwnerRefSetFailed", err.Error())
		_ = r.Status().Patch(ctx, &tj, client.MergeFrom(originalTJ))
		go r.callbackTaskStatus(&tj)
		return ctrl.Result{}, err
	}

	// 7. 检查 Job 是否存在
	found := job.DeepCopyObject().(client.Object)
	err = r.Get(ctx, client.ObjectKeyFromObject(job), found)
	if err != nil {
		if errors.IsNotFound(err) {
			// 创建 Job
			log.Info("Creating new Job", "Namespace", job.GetNamespace(), "Name", job.GetName())
			if err := r.Create(ctx, job); err != nil {
				log.Error(err, "Failed to create Job")
				r.updateStatus(ctx, &tj, "Failed", "JobCreateFailed", err.Error())
				_ = r.Status().Patch(ctx, &tj, client.MergeFrom(originalTJ))
				go r.callbackTaskStatus(&tj)
				return ctrl.Result{}, err
			}
			r.updateStatus(ctx, &tj, "Pending", "JobCreated", "Job created successfully, waiting for pod scheduling")
		} else {
			log.Error(err, "Failed to get Job")
			return ctrl.Result{}, err
		}
	} else {
		// Job已存在，同步状态
		r.syncStatusFromJob(ctx, &tj, found)
	}

	// 8. 同步分布式任务状态
	if err := r.Status().Patch(ctx, &tj, client.MergeFrom(originalTJ)); err != nil {
		log.Error(err, "Failed to patch TrainingJob status")
		return ctrl.Result{}, err
	}

	// 9. 定期调和（5秒后重试）
	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

// buildDistributedPyTorchJob 构建Kubeflow分布式训练任务 PyTorchJob
func (r *TrainingJobReconciler) buildDistributedPyTorchJob(tj *aiv1.TrainingJob) (*trainingv1.PyTorchJob, error) {
	jobName := fmt.Sprintf("%s-pytorch-dist", tj.Name)
	replicaSpecs := map[trainingv1.ReplicaType]*trainingv1.ReplicaSpec{}

	// 通用配置
	backoffLimit := r.buildBackoffLimit(tj)

	// 写入节点选择
	podTemplate := r.buildPodTemplate(tj)
	if tj.Spec.NodeName != "" {
		podTemplate.Spec.NodeName = tj.Spec.NodeName
	}
	// Master节点
	replicaSpecs[trainingv1.PyTorchJobReplicaTypeMaster] = &trainingv1.ReplicaSpec{
		Replicas:      help.Ptr[int32](1),
		RestartPolicy: trainingv1.RestartPolicyOnFailure,
		Template:      podTemplate,
	}

	// Worker 节点 分布式开启
	if tj.Spec.Distributed && tj.Spec.WorkerNum > 0 {
		replicaSpecs[trainingv1.PyTorchJobReplicaTypeWorker] = &trainingv1.ReplicaSpec{
			Replicas:      &tj.Spec.WorkerNum,
			Template:      podTemplate,
			RestartPolicy: trainingv1.RestartPolicyOnFailure,
		}
	}
	return &trainingv1.PyTorchJob{
		ObjectMeta: metav1.ObjectMeta{Name: jobName, Namespace: tj.Namespace},
		Spec: trainingv1.PyTorchJobSpec{
			RunPolicy: trainingv1.RunPolicy{
				BackoffLimit:            &backoffLimit,
				TTLSecondsAfterFinished: &tj.Spec.TTLSecondsAfterFinished,
				ActiveDeadlineSeconds:   &tj.Spec.ActiveDeadlineSeconds,
			},
			PyTorchReplicaSpecs: replicaSpecs,
		},
	}, nil
}

// buildDistributedTFJob 构建Kubeflow分布式训练任务 TFJob
func (r *TrainingJobReconciler) buildDistributedTFJob(tj *aiv1.TrainingJob) (*trainingv1.TFJob, error) {
	jobName := fmt.Sprintf("%s-tf-dist", tj.Name)
	replicaSpecs := map[trainingv1.ReplicaType]*trainingv1.ReplicaSpec{}

	// 通用配置
	backoffLimit := r.buildBackoffLimit(tj)

	podTemplate := r.buildPodTemplate(tj)
	if tj.Spec.NodeName != "" {
		podTemplate.Spec.NodeName = tj.Spec.NodeName
	}

	// Chief 节点（TF 2.x 推荐）
	replicaSpecs[trainingv1.TFJobReplicaTypeMaster] = &trainingv1.ReplicaSpec{
		Replicas:      help.Ptr[int32](1),
		RestartPolicy: trainingv1.RestartPolicyOnFailure,
		Template:      podTemplate,
	}

	// Worker 节点
	if tj.Spec.WorkerNum > 0 {
		replicaSpecs[trainingv1.TFJobReplicaTypeWorker] = &trainingv1.ReplicaSpec{
			Replicas:      &tj.Spec.WorkerNum,
			RestartPolicy: trainingv1.RestartPolicyOnFailure,
			Template:      podTemplate,
		}
	}

	// PS 节点（可选，TF 1.x 架构）
	if tj.Spec.MasterNum > 0 {
		replicaSpecs[trainingv1.TFJobReplicaTypePS] = &trainingv1.ReplicaSpec{
			Replicas:      &tj.Spec.MasterNum,
			RestartPolicy: trainingv1.RestartPolicyOnFailure,
			Template:      podTemplate,
		}
	}

	return &trainingv1.TFJob{
		ObjectMeta: metav1.ObjectMeta{Name: jobName, Namespace: tj.Namespace},
		Spec: trainingv1.TFJobSpec{
			RunPolicy: trainingv1.RunPolicy{
				BackoffLimit:            &backoffLimit,
				TTLSecondsAfterFinished: &tj.Spec.TTLSecondsAfterFinished,
				ActiveDeadlineSeconds:   &tj.Spec.ActiveDeadlineSeconds,
			},
			TFReplicaSpecs: replicaSpecs,
		},
	}, nil

}

// buildSingleNodeJob 构建单节点训练任务 单节点通用 Job（ONNX / 单节点 PyTorch/TF）
func (r *TrainingJobReconciler) buildSingleNodeJob(tj *aiv1.TrainingJob, suffix string) (*batchv1.Job, error) {
	jobName := fmt.Sprintf("%s-%s", tj.Name, suffix)
	backoffLimit := r.buildBackoffLimit(tj)
	container := r.buildContainer(tj)

	podSpec := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyOnFailure,
		Containers:    []corev1.Container{container},
		Volumes:       tj.Spec.Volumes,
	}
	// 【修复】写入节点选择
	if tj.Spec.NodeName != "" {
		podSpec.NodeName = tj.Spec.NodeName
	}

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: jobName, Namespace: tj.Namespace},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &tj.Spec.TTLSecondsAfterFinished,
			ActiveDeadlineSeconds:   &tj.Spec.ActiveDeadlineSeconds,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: r.buildPodMeta(tj),
				Spec:       podSpec,
			},
		},
	}, nil
}

// buildBackoffLimit 构建回退限制
func (r *TrainingJobReconciler) buildBackoffLimit(tj *aiv1.TrainingJob) int32 {
	backoffLimit := int32(3)
	if tj.Spec.BackoffLimit > 0 {
		backoffLimit = tj.Spec.BackoffLimit
	}
	return backoffLimit
}

// buildContainer 构建容器
func (r *TrainingJobReconciler) buildContainer(tj *aiv1.TrainingJob) corev1.Container {
	container := corev1.Container{
		Name:         "trainer-" + help.RandomString(10),
		Image:        tj.Spec.Image,
		Command:      tj.Spec.Command,
		Env:          tj.Spec.Env,
		VolumeMounts: tj.Spec.VolumeMounts,
		Resources:    r.buildResources(tj.Spec.Resources),
	}

	// 监控参数去重追加
	if tj.Spec.EnableMonitor {
		monitorArgs := []string{"--monitor-enabled=true", "--monitor-endpoint=http://monitor-agent.kubeai-system.svc:9090"}
		argMap := make(map[string]bool)
		for _, arg := range container.Args {
			argMap[arg] = true
		}
		for _, arg := range monitorArgs {
			if !argMap[arg] {
				container.Args = append(container.Args, arg)
			}
		}
	}
	return container
}

// buildPodTemplate 构建 Pod 模板
func (r *TrainingJobReconciler) buildPodTemplate(tj *aiv1.TrainingJob) corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		ObjectMeta: r.buildPodMeta(tj),
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{r.buildContainer(tj)},
			Volumes:    tj.Spec.Volumes,
		},
	}
}

// buildPodMeta 构建 Pod 元数据
func (r *TrainingJobReconciler) buildPodMeta(tj *aiv1.TrainingJob) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Labels:      r.buildPodLabels(tj), // 注入日志/监控标签
		Annotations: r.buildMonitorAnnotations(tj.Spec.EnableMonitor),
	}
}

// buildResources 构建资源限制
func (r *TrainingJobReconciler) buildResources(res aiv1.ResourceRequirements) corev1.ResourceRequirements {
	reqList := corev1.ResourceList{}
	limitList := corev1.ResourceList{}

	if res.CPU != "" {
		reqList[corev1.ResourceCPU] = resource.MustParse(res.CPU)
		limitList[corev1.ResourceCPU] = resource.MustParse(res.CPU)
	}
	if res.Memory != "" {
		reqList[corev1.ResourceMemory] = resource.MustParse(res.Memory)
		limitList[corev1.ResourceMemory] = resource.MustParse(res.Memory)
	}
	// GPU：Requests与Limits保持一致，符合K8s GPU调度规范
	if res.GPU != "" && strings.TrimSpace(res.GPU) != "0" {
		gpuQuantity := resource.MustParse(res.GPU)
		reqList["nvidia.com/gpu"] = gpuQuantity
		limitList["nvidia.com/gpu"] = gpuQuantity
	}

	return corev1.ResourceRequirements{Requests: reqList, Limits: limitList}
}

// updateStatus
// 更新 TrainingJob CR 的 phase、condition、状态信息
// 标准化云原生状态流转
func (r *TrainingJobReconciler) updateStatus(ctx context.Context, tj *aiv1.TrainingJob, phase, reason, msg string) {
	log := log.FromContext(ctx)
	now := metav1.Now()

	// 1. 检查状态是否真的变化，避免不必要的更新
	if tj.Status.Phase == phase && tj.Status.Reason == reason && tj.Status.Message == msg {
		return
	}

	// 2. 更新基础状态
	tj.Status.Phase = phase
	tj.Status.Reason = reason
	tj.Status.Message = msg
	tj.Status.LastTransitionTime = now

	// 3. 管理 Conditions（追加或更新，而非覆盖）
	r.setCondition(tj, ConditionTypeJobReady, getConditionStatus(phase, "Ready"), reason, msg, now)
	r.setCondition(tj, ConditionTypeJobSucceeded, getConditionStatus(phase, "Succeeded"), reason, msg, now)
	r.setCondition(tj, ConditionTypeJobFailed, getConditionStatus(phase, "Failed"), reason, msg, now)

	log.Info("Status updated", "phase", phase, "reason", reason)
}

// setCondition 添加或更新 Condition
func (r *TrainingJobReconciler) setCondition(tj *aiv1.TrainingJob, condType string, status metav1.ConditionStatus, reason, msg string, now metav1.Time) {
	// 查找是否已存在该 Condition
	var existingIdx *int
	for i, cond := range tj.Status.Conditions {
		if cond.Type == condType {
			existingIdx = &i
			break
		}
	}

	newCond := metav1.Condition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            msg,
		ObservedGeneration: tj.Generation, // 关键：关联 CR 的 Generation
	}

	if existingIdx != nil {
		// 已存在：检查是否真的变化
		oldCond := tj.Status.Conditions[*existingIdx]
		if oldCond.Status == newCond.Status && oldCond.Reason == newCond.Reason && oldCond.Message == newCond.Message {
			return // 无变化，不更新
		}
		// 只有状态变化时才更新 LastTransitionTime
		if oldCond.Status == newCond.Status {
			newCond.LastTransitionTime = oldCond.LastTransitionTime
		}
		tj.Status.Conditions[*existingIdx] = newCond
	} else {
		// 不存在：追加
		tj.Status.Conditions = append(tj.Status.Conditions, newCond)
	}
}

// getConditionStatus 根据 Phase 映射 ConditionStatus
func getConditionStatus(phase, condType string) metav1.ConditionStatus {
	switch phase {
	case "Pending":
		if condType == "Ready" {
			return metav1.ConditionFalse
		}
		return metav1.ConditionUnknown
	case "Running":
		if condType == "Ready" {
			return metav1.ConditionTrue
		}
		return metav1.ConditionUnknown
	case "Succeeded":
		if condType == "Succeeded" {
			return metav1.ConditionTrue
		}
		if condType == "Failed" {
			return metav1.ConditionFalse
		}
		return metav1.ConditionTrue
	case "Failed":
		if condType == "Failed" {
			return metav1.ConditionTrue
		}
		if condType == "Succeeded" {
			return metav1.ConditionFalse
		}
		return metav1.ConditionFalse
	default:
		return metav1.ConditionUnknown
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *TrainingJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&aiv1.TrainingJob{}).
		Owns(&trainingv1.PyTorchJob{}).
		Owns(&trainingv1.TFJob{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}

// buildPodLabels
// 给训练 Pod 打标签，用于：
// 1. 日志检索（Loki）
// 2. 监控指标（Prometheus）
// 3. 服务发现
func (r *TrainingJobReconciler) buildPodLabels(tj *aiv1.TrainingJob) map[string]string {
	labels := map[string]string{
		"app":                    "kubeai-training-job",
		"training.job.name":      tj.Name,
		"training.job.namespace": tj.Namespace,
		"training.framework":     tj.Spec.Framework,
		"training.distributed":   strconv.FormatBool(tj.Spec.Distributed),
	}

	if tj.Spec.ModelID != "" {
		labels["training.model.id"] = tj.Spec.ModelID
	}
	if tj.Spec.EnableLogs {
		labels["training.logs.enabled"] = strconv.FormatBool(tj.Spec.EnableLogs)
	}
	return labels
}

// buildMonitorAnnotations
// 自动注入 Prometheus 监控注解
// 开启后 Prometheus 自动抓取训练指标（loss/epoch/accuracy）
func (r *TrainingJobReconciler) buildMonitorAnnotations(enableMonitor bool) map[string]string {
	if !enableMonitor {
		return nil
	}
	return map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/port":   "8082",
		"prometheus.io/path":   "/metrics",
		"prometheus.io/job":    "training-job",
	}
}

// buildMonitorArgs
// 向容器注入监控相关启动参数
func (r *TrainingJobReconciler) buildMonitorArgs(enableMonitor bool) []string {
	if !enableMonitor {
		return nil
	}
	return []string{"--monitor-enabled=true", "--monitor-endpoint=http://monitor-agent.kubeai-system.svc:9090"}
}

// syncStatusFromJob 从下游 Job 同步状态到 TrainingJob
func (r *TrainingJobReconciler) syncStatusFromJob(ctx context.Context, tj *aiv1.TrainingJob, job client.Object) {
	switch j := job.(type) {
	case *trainingv1.PyTorchJob:
		r.syncKubeflowJobStatus(ctx, tj, j.Status.Conditions)
	case *trainingv1.TFJob:
		r.syncKubeflowJobStatus(ctx, tj, j.Status.Conditions)
	case *batchv1.Job:
		r.syncBatchJobStatus(ctx, tj, j.Status)
	}
}

// syncKubeflowJobStatus 同步 Kubeflow Job（PyTorch/TF）的状态
func (r *TrainingJobReconciler) syncKubeflowJobStatus(
	ctx context.Context,
	tj *aiv1.TrainingJob,
	jobConditions []trainingv1.JobCondition,
) {
	for _, cond := range jobConditions {
		if cond.Status != corev1.ConditionTrue {
			continue
		}
		switch cond.Type {
		case trainingv1.JobCreated:
			r.updateStatus(ctx, tj, "Pending", "JobCreated", "Job has been created")
		case trainingv1.JobRunning:
			r.updateStatus(ctx, tj, "Running", "JobRunning", "Job is running")
		case trainingv1.JobSucceeded:
			r.updateStatus(ctx, tj, "Succeeded", "JobSucceeded", "Job completed successfully")
		case trainingv1.JobFailed:
			r.updateStatus(ctx, tj, "Failed", "JobFailed", fmt.Sprintf("Job failed: %s", cond.Message))
		}
	}
}

// syncBatchJobStatus 同步普通 Batch Job 的状态
func (r *TrainingJobReconciler) syncBatchJobStatus(
	ctx context.Context,
	tj *aiv1.TrainingJob,
	jobStatus batchv1.JobStatus,
) {
	switch {
	case jobStatus.Succeeded > 0:
		r.updateStatus(ctx, tj, "Succeeded", "JobSucceeded", "Job completed successfully")
	case jobStatus.Failed > 0:
		r.updateStatus(ctx, tj, "Failed", "JobFailed", "Job failed to complete")
	case jobStatus.Active > 0:
		r.updateStatus(ctx, tj, "Running", "JobRunning", "Job is active")
	default:
		r.updateStatus(ctx, tj, "Pending", "JobPending", "Job is pending")
	}
}

// isModelRegistered 检查模型是否已注册
func (r *TrainingJobReconciler) isModelRegistered(tj *aiv1.TrainingJob) bool {
	for _, cond := range tj.Status.Conditions {
		if cond.Type == ConditionTypeModelRegistered && cond.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

// registerTrainedModel 适配 model-manager 接口的模型注册逻辑
func (r *TrainingJobReconciler) registerTrainedModel(ctx context.Context, tj *aiv1.TrainingJob) error {
	log := log.FromContext(ctx)

	// 前置校验：仅训练成功且有输出路径时注册
	if tj.Status.Phase != "Succeeded" {
		return fmt.Errorf("only succeeded job can register model, current phase: %s", tj.Status.Phase)
	}
	if tj.Spec.OutputPath == "" {
		return fmt.Errorf("model output path is empty, skip register")
	}

	// 构造 model-manager
	registerReq := &callbackClient.ModelRegisterRequest{
		ModelName:   tj.Spec.ModelName,
		Version:     fmt.Sprintf("v%d", time.Now().Unix()), // 自动生成版本号
		StoragePath: tj.Spec.OutputPath,                    // 训练输出路径
		Framework:   tj.Spec.Framework,
		TaskType:    "training",
		Description: fmt.Sprintf("trained model from job %s", tj.Name),
		Metadata: map[string]string{
			"training_job_name": tj.Name,
			"namespace":         tj.Namespace,
			"distributed":       strconv.FormatBool(tj.Spec.Distributed),
		},
		TrainingJobName: tj.Name,
		Namespace:       tj.Namespace,
		ModelID:         tj.Spec.ModelID, // 关联已有模型ID（如有）
	}

	// 调用 model-manager 注册接口
	resp, err := r.ModelClient.RegisterModel(ctx, registerReq)
	if err != nil {
		return fmt.Errorf("call model-manager RegisterModel failed: %w", err)
	}

	// 校验响应
	if !resp.Success {
		return fmt.Errorf("model-manager register failed: %s (model_id: %s)", resp.Message, resp.ModelID)
	}

	// 更新 TrainingJob 状态（记录模型ID）
	tj.Spec.ModelID = resp.ModelID
	tj.Status.RegisteredModelID = resp.ModelID
	log.Info("Model registered successfully", "model_id", resp.ModelID, "job_name", tj.Name)
	return nil
}

// callbackTaskStatus 回调任务侧状态
func (r *TrainingJobReconciler) callbackTaskStatus(tj *aiv1.TrainingJob) {
	if r.JobScheduleAddr == "" {
		logx.Infof("callback task status to business side, no callback url, task name: %s", tj.Name)
		return
	}

	log := logx.WithContext(context.Background()).WithFields(
		logx.Field("task_id", tj.Name),
		logx.Field("phase", tj.Status.Phase),
		logx.Field("jobScheduleAddr", r.JobScheduleAddr),
	)

	// 1. 构建回调请求体
	reqBody := &callbackClient.CallBackTaskStatusReq{
		TaskID:     tj.Name,
		ModelName:  tj.Spec.ModelName,
		Phase:      tj.Status.Phase,
		Reason:     tj.Status.Reason,
		Message:    tj.Status.Message,
		ModelID:    tj.Status.RegisteredModelID,
		NodeName:   tj.Spec.NodeName,
		FinishedAt: time.Now().Format(time.RFC3339),
	}

	// 2. 指数退避重试配置
	maxRetries := 3
	baseDelay := 1 * time.Second
	maxDelay := 10 * time.Second

	// 3. 重试循环
	for i := 0; i < maxRetries; i++ {
		// 计算延迟时间：指数退避
		delay := time.Duration(math.Pow(2, float64(i))) * baseDelay
		if delay > maxDelay {
			delay = maxDelay
		}

		// 带超时的HTTP请求
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		resp, err := r.JobScheduleClient.CallBackTaskStatus(ctx, reqBody)
		if err != nil {
			log.Errorf("create callback request failed (attempt %d/%d): %v", i+1, maxRetries, err)
			cancel()
			time.Sleep(delay)
			continue
		}
		cancel() // 立即释放资源

		if err != nil {
			log.Errorf("send callback request failed (attempt %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(delay)
			continue
		}

		// 检查响应状态码
		if resp.Code == http.StatusOK {
			log.Infof("callback success (attempt %d/%d), status code: %d", i+1, maxRetries, resp.Code)
			return
		}

		log.Errorf("callback failed (attempt %d/%d), status code: %d", i+1, maxRetries, resp.Code)
		if i < maxRetries-1 {
			time.Sleep(delay)
		}
	}

	// 所有重试均失败
	log.Errorf("all callback attempts failed, task: %s", tj.Name)

}
