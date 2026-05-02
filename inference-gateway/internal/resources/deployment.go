package resources

import (
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	aiv1 "kubeai-inference-gateway/inferenceservice/api/v1"
	modelmanager "kubeai-inference-gateway/internal/client"
	"strconv"
)

const (
	LabelKeyApp       = "app.kubernetes.io/name"
	LabelKeyVersion   = "app.kubernetes.io/version"
	LabelKeyRole      = "app.kubernetes.io/component"
	LabelManagedBy    = "app.kubernetes.io/managed-by"
	LabelKeyFramework = "app.kubernetes.io/framework"
)

// ConvertEnvVars 构建环境变量列表
func ConvertEnvVars(meta *modelmanager.ModelMetadata, modelManagerAddr string) []corev1.EnvVar {
	envVars := []corev1.EnvVar{
		// 注入 K8s  downward API 环境变量
		{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
		}},
		{Name: "NODE_NAME", ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"},
		}},
		{Name: "ENV", Value: "production"},
	}

	// 如果获取到了模型元数据，注入环境变量
	if meta != nil {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "MODEL_STORAGE_PATH",
			Value: meta.StoragePath,
		}, corev1.EnvVar{
			Name:  "MODEL_NAME",
			Value: meta.ModelName,
		}, corev1.EnvVar{
			Name:  "MODEL_VERSION",
			Value: meta.ModelVersion,
		}, corev1.EnvVar{
			Name:  "MODEL_MANAGER_ADDR",
			Value: modelManagerAddr,
		},
		)
	}
	return envVars
}

// ConvertInitContainers 构建初始化容器列表
func ConvertInitContainers(modelManagerAddr string, meta *modelmanager.ModelMetadata) []corev1.Container {
	// 初始化容器，用于模型下载
	initContainer := corev1.Container{
		Name:    "model-downloader",
		Image:   "curlimages/curl:latest",
		Command: []string{"/bin/sh", "-c"},
		Args: []string{
			fmt.Sprintf(`
				mkdir -p /models && \
				curl -f -o /models/model.bin %s/api/v1/models/%s/versions/%s/download
				if [ $? -ne 0 ]; then
					echo "ERROR: 模型拉取失败"
					exit 1
				fi
				echo "SUCCESS: 模型拉取完成"
			`, modelManagerAddr, meta.ModelName, meta.ModelVersion),
		},
		VolumeMounts: []corev1.VolumeMount{{Name: "model-storage", MountPath: "/models"}},
	}
	return []corev1.Container{initContainer}
}

// ConvertVolumes 构建共享存储卷列表
func ConvertVolumes() []corev1.Volume {
	// 共享存储卷
	volumes := []corev1.Volume{{
		Name:         "model-storage",
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	}}
	return volumes
}

// ConvertLifecycle 构建生命周期事件处理函数
func ConvertLifecycle() *corev1.Lifecycle {
	lifecycle := &corev1.Lifecycle{
		PostStart: &corev1.LifecycleHandler{
			Exec: &corev1.ExecAction{
				Command: []string{"/bin/sh", "-c", `
				# 1. 等待metrics接口启动（10秒超时）
				timeout=10
				for i in $(seq $timeout); do
					if curl -s -f http://127.0.0.1:8501/metrics > /dev/null; then
						echo "metrics server started successfully"
						break
					fi
					echo "waiting for metrics server... $i/$timeout"
					sleep 1
				done

				# 2. 向模型管理中心上报：推理实例已启动就绪
				curl -s -X POST ${MODEL_MANAGER_ADDR%/*}/api/v1/models/version/status \
					-H "Content-Type: application/json" \
					-d '{
						"model_name":"'"${MODEL_NAME}"'",
						"model_version":"'"${MODEL_VERSION}"'",
						"status":"active",
						"instance_status":"healthy",
						"instance":"'"${POD_NAME}"'",
						"node":"'"${NODE_NAME}"'"
						"message":"startup"
					}'

				echo "instance registered to model manager"
				`},
			},
		},
		PreStop: &corev1.LifecycleHandler{
			Exec: &corev1.ExecAction{
				Command: []string{"/bin/sh", "-c", `
				curl -s -X POST ${MODEL_MANAGER_ADDR%/*}/api/v1/models/version/status \
					-H "Content-Type: application/json" \
					-d '{
						"model_name":"'"${MODEL_NAME}"'",
						"model_version":"'"${MODEL_VERSION}"'",
						"status":"inactive",
						"instance_status":"unhealthy",
						"instance":"'"${POD_NAME}"'",
						"message":"shutdown"
					}'
				echo "instance offline reported"
			   `},
			},
		},
	}
	return lifecycle
}

// ConvertLivenessProbe 构建 livenessProbe
func ConvertLivenessProbe(port int32) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/api/v/inference/health", Port: intstr.FromInt(int(port)),
			},
		},
		InitialDelaySeconds: 10,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		FailureThreshold:    3,
		TimeoutSeconds:      5,
	}
}

// ConvertReadinessProbe 构建 readinessProbe
func ConvertReadinessProbe(port int32) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/api/v/inference/ready", Port: intstr.FromInt(int(port)),
			},
		},
		InitialDelaySeconds: 10,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		FailureThreshold:    3,
		TimeoutSeconds:      5,
	}
}

// ConvertStableContainers 构建稳定版容器列表
func ConvertStableContainers(isvc *aiv1.InferenceService, meta *modelmanager.ModelMetadata, modelManagerAddr string) []corev1.Container {
	return []corev1.Container{
		{
			Name:  fmt.Sprintf("%s-%s", isvc.Spec.ModelName, isvc.Spec.ModelVersion),
			Image: getImage(isvc, meta.Framework),
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: getContainerPort(isvc),
					Name:          "http",
					Protocol:      corev1.ProtocolTCP,
				}, // 默认推理端口
			},
			Resources:      isvc.Spec.Resources,
			Env:            ConvertEnvVars(meta, modelManagerAddr), // 【关键】注入环境变量
			LivenessProbe:  ConvertLivenessProbe(isvc.Spec.Service.Port),
			ReadinessProbe: ConvertReadinessProbe(isvc.Spec.Service.Port),
			Lifecycle:      ConvertLifecycle(),
		},
	}
}

// ConvertCanaryContainers 构建灰度版容器列表
func ConvertCanaryContainers(isvc *aiv1.InferenceService, meta *modelmanager.ModelMetadata, modelManagerAddr string) []corev1.Container {
	return []corev1.Container{
		{
			Name:  fmt.Sprintf("%s-%s", isvc.Spec.ModelName, isvc.Spec.ModelVersion),
			Image: getCanaryImage(isvc, meta.Framework),
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: getContainerPort(isvc),
					Name:          "http",
					Protocol:      corev1.ProtocolTCP,
				},
			},
			Resources: func() corev1.ResourceRequirements {
				if isvc.Spec.Canary.Resources != nil {
					return *isvc.Spec.Canary.Resources
				}
				return isvc.Spec.Resources
			}(),
			Env:            ConvertEnvVars(meta, modelManagerAddr), // 【关键】注入环境变量
			LivenessProbe:  ConvertLivenessProbe(isvc.Spec.Service.Port),
			ReadinessProbe: ConvertReadinessProbe(isvc.Spec.Service.Port),
			// 推理指标上报（Prometheus）
			Lifecycle: ConvertLifecycle(),
		},
	}
}

// ConvertLabels 构建标签
func ConvertLabels(isvc *aiv1.InferenceService, meta *modelmanager.ModelMetadata) map[string]string {
	labels := make(map[string]string)
	labels[LabelKeyApp] = isvc.Name
	labels[LabelKeyVersion] = meta.ModelVersion
	labels[LabelKeyFramework] = meta.Framework
	labels[LabelManagedBy] = "kubeai-platform"
	if isvc.Spec.Canary != nil && isvc.Spec.Canary.Enabled {
		labels[LabelKeyRole] = "canary"
	} else {
		labels[LabelKeyRole] = "stable"
	}
	return labels
}

// NewStableDeployment 创建稳定版 Deployment
func NewStableDeployment(isvc *aiv1.InferenceService, meta *modelmanager.ModelMetadata, modelManagerAddr string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      isvc.Name + "-stable",
			Namespace: isvc.Namespace,
			Labels:    ConvertLabels(isvc, meta),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(isvc, aiv1.GroupVersion.WithKind("InferenceService")),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: getReplicas(isvc),
			Selector: &metav1.LabelSelector{
				MatchLabels: ConvertLabels(isvc, meta),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ConvertLabels(isvc, meta),
					Annotations: map[string]string{
						"prometheus.io/scrape": "true",
						"prometheus.io/port":   strconv.Itoa(int(getContainerPort(isvc))),
						"prometheus.io/path":   "/api/v1/inference/metrics",
					},
				},
				Spec: corev1.PodSpec{
					InitContainers: ConvertInitContainers(modelManagerAddr, meta),
					Containers:     ConvertStableContainers(isvc, meta, modelManagerAddr),
					Volumes:        ConvertVolumes(),
				},
			},
		},
	}
}

// NewCanaryDeployment 创建灰度版 Deployment
func NewCanaryDeployment(isvc *aiv1.InferenceService, meta *modelmanager.ModelMetadata, modelManagerAddr string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      isvc.Name + "-canary",
			Namespace: isvc.Namespace,
			Labels:    ConvertLabels(isvc, meta),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(isvc, aiv1.GroupVersion.WithKind("InferenceService")),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: getReplicas(isvc),
			Selector: &metav1.LabelSelector{
				MatchLabels: ConvertLabels(isvc, meta),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ConvertLabels(isvc, meta),
					Annotations: map[string]string{
						"prometheus.io/scrape": "true",
						"prometheus.io/port":   strconv.Itoa(int(getContainerPort(isvc))),
						"prometheus.io/path":   "/api/v1/inference/metrics",
					},
				},
				Spec: corev1.PodSpec{
					InitContainers: ConvertInitContainers(modelManagerAddr, meta),
					Containers:     ConvertCanaryContainers(isvc, meta, modelManagerAddr),
					Volumes:        ConvertVolumes(),
				},
			},
		},
	}
}
