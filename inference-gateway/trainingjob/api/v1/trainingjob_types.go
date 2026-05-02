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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TrainingJobSpec defines the desired state of TrainingJob
type TrainingJobSpec struct {
	// 深度学习框架 pytorch/tensorflow/onnx
	// +kubebuilder:validation:Enum=pytorch;tensorflow;onnx
	// +kubebuilder:default=pytorch
	Framework string `json:"framework"`
	// 容器镜像
	// +kubebuilder:validation:MinLength=1
	Image string   `json:"image"`
	Args  []string `json:"args,omitempty"`
	// 启动命令
	Command []string `json:"command,omitempty"`

	// 分布式训练核心配置
	Distributed bool  `json:"distributed,omitempty"` // 是否开启分布式
	WorkerNum   int32 `json:"workerNum,omitempty"`   // 工作节点数
	MasterNum   int32 `json:"masterNum,omitempty"`   //Master 节点数

	// 计算资源
	// +kubebuilder:validation:Required
	Resources ResourceRequirements `json:"resources,omitempty"`
	// 环境变量
	Env []corev1.EnvVar `json:"env,omitempty"`
	// 挂载卷
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`
	Volumes      []corev1.Volume      `json:"volumes,omitempty"`
	// 数据集路径
	DatasetPath string `json:"datasetPath,omitempty"`
	// 训练输出路径
	// +kubebuilder:validation:MinLength=1
	OutputPath string `json:"outputPath"`
	// 失败重试次数
	// +kubebuilder:default=3
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=10
	BackoffLimit int32 `json:"backoffLimit,omitempty"`
	// 完成后自动清理时间（秒）
	// +kubebuilder:default=3600
	// +kubebuilder:validation:Minimum=60
	TTLSecondsAfterFinished int32 `json:"ttlSecondsAfterFinished,omitempty"`
	// 任务最大运行时间（秒），超时自动终止
	// +kubebuilder:default=86400
	// +kubebuilder:validation:Minimum=60
	ActiveDeadlineSeconds int64 `json:"activeDeadlineSeconds,omitempty"`
	// 关联模型ID (对接业务)
	ModelID   string `json:"modelID,omitempty"`
	ModelName string `json:"modelName,omitempty"`

	// 监控/日志开关
	EnableMonitor bool `json:"enableMonitor,omitempty"`
	EnableLogs    bool `json:"enableLogging,omitempty"`
	// 调度节点名称（由调度中心填充）
	NodeName string `json:"nodeName,omitempty"`
	// 节点选择器
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// 容忍器
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// ResourceRequirements 资源定义
type ResourceRequirements struct {
	// +kubebuilder:validation:Pattern=`^[0-9]+m?$`
	CPU string `json:"cpu,omitempty"`
	// +kubebuilder:validation:Pattern=`^[0-9]+[KMG]i?$`
	Memory string `json:"memory,omitempty"`
	// +kubebuilder:validation:Pattern=`^[0-9]+[KMG]i?$`
	GPU string `json:"gpu,omitempty"`
}

// TrainingJobStatus defines the observed state of TrainingJob.
type TrainingJobStatus struct {
	// Phase is the current phase of the training job (Pending/Running/Succeeded/Failed)
	Phase string `json:"phase,omitempty"`
	// Reason is a brief reason for the current phase
	Reason string `json:"reason,omitempty"`
	// Message is a human-readable message explaining the current phase
	Message string `json:"message,omitempty"`
	// LastTransitionTime is the last time the status transitioned
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// Conditions represents the latest available observations of the job's current state
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// 训练完成后注册的模型ID
	RegisteredModelID string `json:"registeredModelID,omitempty"`
	// 模型注册重试次数
	ModelRegisterRetryCount int32 `json:"modelRegisterRetryCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Model",type="string",JSONPath=".spec.modelName"
// +kubebuilder:printcolumn:name="Framework",type="string",JSONPath=".spec.framework"
// +kubebuilder:printcolumn:name="Distributed",type="boolean",JSONPath=".spec.distributed"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:shortName=tj
// TrainingJob is the Schema for the trainingjobs API
type TrainingJob struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec defines the desired state of TrainingJob
	// +required
	Spec TrainingJobSpec `json:"spec"`

	// status defines the observed state of TrainingJob
	// +optional
	Status TrainingJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// TrainingJobList contains a list of TrainingJob
type TrainingJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TrainingJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TrainingJob{}, &TrainingJobList{})
}
