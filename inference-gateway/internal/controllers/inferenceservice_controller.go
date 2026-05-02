package controller

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	modelClient "kubeai-inference-gateway/internal/client"
	"kubeai-inference-gateway/internal/help"
	"kubeai-inference-gateway/internal/model"
	"kubeai-inference-gateway/internal/resources"
	"kubeai-inference-gateway/pkg/metrics"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"

	aiv1 "kubeai-inference-gateway/inferenceservice/api/v1"
)

const InferenceServiceFinalizer = "ai.kubeai.io/finalizer"

type InferenceServiceReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	ModelManagerAddr  string                          // 模型服务地址
	JobScheduleAddr   string                          // 任务调度服务地址
	ModelClient       *modelClient.ModelManagerClient //客户端实例
	JobScheduleClient *modelClient.JobScheduleClient  //客户端实例
}

//+kubebuilder:rbac:groups=ai.kubeai.io,resources=inferenceservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ai.kubeai.io,resources=inferenceservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ai.kubeai.io,resources=inferenceservices/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.

// Reconcile 实现协调循环
func (r *InferenceServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("inferenceService", req.NamespacedName)

	// 1. 获取 InferenceService 实例
	isvc := &aiv1.InferenceService{}
	if err := r.Get(ctx, req.NamespacedName, isvc); err != nil {
		log.Error(err, "unable to fetch InferenceService")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	old := isvc.DeepCopy()

	// 2. Finalizer
	if isvc.DeletionTimestamp.IsZero() {
		if !help.ContainsString(isvc.Finalizers, InferenceServiceFinalizer) {
			isvc.Finalizers = append(isvc.Finalizers, InferenceServiceFinalizer)
			if err := r.Update(ctx, isvc); err != nil {
				log.Error(err, "Failed to add finalizer to InferenceService")
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
	} else {
		if help.ContainsString(isvc.Finalizers, InferenceServiceFinalizer) {
			log.Info("start cleaning up InferenceService resources")
			// 删除时清理所有关联资源，避免残留
			_ = r.cleanupCanaryResources(ctx, isvc)
			_ = r.cleanupStableResources(ctx, isvc)
			isvc.Finalizers = help.RemoveString(isvc.Finalizers, InferenceServiceFinalizer)
			if err := r.Update(ctx, isvc); err != nil {
				log.Error(err, "Failed to remove finalizer from InferenceService")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// 3. 超时检查
	if isvc.Spec.ActiveDeadline > 0 {
		elapsed := time.Since(isvc.CreationTimestamp.Time).Seconds()
		if elapsed > float64(isvc.Spec.ActiveDeadline) {
			r.setStatus(isvc, "Failed", "DeadlineExceeded", "timeout")
			_ = r.Status().Patch(ctx, isvc, client.MergeFrom(old))
			return ctrl.Result{}, nil
		}
	}

	// 4. 调用 Model Manager 获取模型路径
	log.Info("Fetching model metadata", "ModelName", isvc.Spec.ModelName, "Version", isvc.Spec.ModelVersion)
	var meta *modelClient.ModelMetadata
	if r.ModelClient != nil && isvc.Spec.Image == "" {
		var err error
		meta, err = r.ModelClient.GetModelMetadata(ctx, isvc.Spec.ModelName, isvc.Spec.ModelVersion)
		if err != nil {
			r.setStatus(isvc, "ModelError", "FetchFailed", err.Error())
			_ = r.Status().Patch(ctx, isvc, client.MergeFrom(old))
			return ctrl.Result{RequeueAfter: 5 * time.Second}, err
		}
	}

	// 5. 协调 Stable Deployment
	stableDeploy := resources.NewStableDeployment(isvc, meta, r.ModelManagerAddr)
	if err := r.reconcileDeployment(ctx, isvc, stableDeploy); err != nil {
		r.setStatus(isvc, "Deploying", "StableFailed", err.Error())
		_ = r.Status().Patch(ctx, isvc, client.MergeFrom(old))
		log.Error(err, "Failed to reconcile Stable Deployment")
		return ctrl.Result{}, err
	}

	// 6. 协调 Stable Service
	stableSvc := resources.NewStableService(isvc)
	if err := r.reconcileService(ctx, isvc, stableSvc); err != nil {
		r.setStatus(isvc, "Deploying", "StableFailed", err.Error())
		_ = r.Status().Patch(ctx, isvc, client.MergeFrom(old))
		log.Error(err, "Failed to reconcile Stable Service")
		return ctrl.Result{}, err
	}

	// 7. 协调 Canary 资源
	if isvc.Spec.Canary != nil && isvc.Spec.Canary.Enabled {
		// 初始化 Canary 状态为 Pending
		if isvc.Status.CanaryState == "" {
			isvc.Status.CanaryState = string(model.StatusPending)
		}
		if err := r.Status().Update(ctx, isvc); err != nil {
			log.Error(err, "Failed to update InferenceService status")
			return ctrl.Result{}, err
		}

		// Canary Deployment
		canaryDeploy := resources.NewCanaryDeployment(isvc, meta, r.ModelManagerAddr)
		if err := r.reconcileDeployment(ctx, isvc, canaryDeploy); err != nil {
			log.Error(err, "Failed to reconcile Canary Deployment")
			return ctrl.Result{}, err
		}
		// Canary Service
		canarySvc := resources.NewCanaryService(isvc)
		if err := r.reconcileService(ctx, isvc, canarySvc); err != nil {
			log.Error(err, "Failed to reconcile Canary Service")
			return ctrl.Result{}, err
		}
		// Canary Ingress
		canaryIng := resources.NewCanaryIngress(isvc)
		if err := r.reconcileIngress(ctx, isvc, canaryIng); err != nil {
			log.Error(err, "Failed to reconcile Canary Ingress")
			return ctrl.Result{}, err
		}

	} else {
		if err := r.cleanupCanaryResources(ctx, isvc); err != nil {
			log.Error(err, "Failed to cleanup canary resources")
			return ctrl.Result{}, err
		}
	}

	// 8. 协调主 Ingress
	mainIng := resources.NewIngress(isvc)
	if err := r.reconcileIngress(ctx, isvc, mainIng); err != nil {
		log.Error(err, "Failed to reconcile main Ingress")
		return ctrl.Result{}, err
	}

	// 9. 协调 HPA
	hpa := resources.NewHPA(isvc)
	if err := r.reconcileHPA(ctx, isvc, hpa); err != nil {
		log.Error(err, "Failed to reconcile HPA")
		return ctrl.Result{}, err
	} else {
		// 如果没有配置 Autoscaling，删除已存在的 HPA
		existingHPA := &autoscalingv2.HorizontalPodAutoscaler{}
		err := r.Get(ctx, types.NamespacedName{Name: isvc.Name, Namespace: isvc.Namespace}, existingHPA)
		if err == nil {
			log.Info("Deleting existing HPA as Autoscaling is not configured")
			if err := r.Delete(ctx, existingHPA); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	isvc.Status.Ready = true
	isvc.Status.URL = fmt.Sprintf("http://%s.%s.svc.cluster.local", isvc.Name, isvc.Namespace)

	// Update metrics
	r.updateMetrics(isvc)

	// 10. 更新 Status
	if err := r.updateStatus(ctx, isvc); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// reconcileDeployment 通用的 Deployment 调和函数
func (r *InferenceServiceReconciler) reconcileDeployment(ctx context.Context, isvc *aiv1.InferenceService, desired *appsv1.Deployment) error {
	log := log.FromContext(ctx)
	if desired == nil {
		return nil
	}
	found := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Deployment", "Deployment.Namespace", desired.Namespace, "Deployment.Name", desired.Name)
		if err := r.Create(ctx, desired); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		// 更新逻辑：这里简化处理，实际中可能需要更精细的 Patch 策略 深度比较以避免不必要的更新
		if equality.Semantic.DeepEqual(found.Spec.Template.Spec, desired.Spec.Template.Spec) ||
			equality.Semantic.DeepEqual(found.Spec.Replicas, desired.Spec.Replicas) {
			log.Info("No changes detected in Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return nil
		}
		log.Info("Updating Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
		// 保留 ResourceVersion
		desired.ResourceVersion = found.ResourceVersion
		desired.Spec.Replicas = found.Spec.Replicas // 保留 HPA 调整后的副本数
		if err := r.Update(ctx, desired); err != nil {
			return err
		}
	}
	return nil
}

// reconcileService 通用的 Service 调和函数
func (r *InferenceServiceReconciler) reconcileService(ctx context.Context, isvc *aiv1.InferenceService, desired *corev1.Service) error {
	log := log.FromContext(ctx)
	if desired == nil {
		return nil
	}
	found := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Service", "Service.Namespace", desired.Namespace, "Service.Name", desired.Name)
		if err := r.Create(ctx, desired); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		if equality.Semantic.DeepEqual(found.Spec.Selector, desired.Spec.Selector) ||
			equality.Semantic.DeepEqual(found.Spec.Ports, desired.Spec.Ports) ||
			equality.Semantic.DeepEqual(found.ObjectMeta.Annotations, desired.ObjectMeta.Annotations) {
			log.Info("No changes detected in Service Selector", "Service.Namespace", found.Namespace, "Service.Name", found.Name)
			return nil
		}
		// Service 的 ClusterIP 是 immutable的，更新时需要注意
		log.Info("Updating Service", "Service.Namespace", found.Namespace, "Service.Name", found.Name)
		desired.ResourceVersion = found.ResourceVersion
		desired.Spec.ClusterIP = found.Spec.ClusterIP
		if err := r.Update(ctx, desired); err != nil {
			return err
		}
	}
	return nil
}

// reconcileIngress 通用的 Ingress 调和函数
func (r *InferenceServiceReconciler) reconcileIngress(ctx context.Context, isvc *aiv1.InferenceService, desired *networkingv1.Ingress) error {
	log := log.FromContext(ctx)
	if desired == nil {
		return nil
	}

	found := &networkingv1.Ingress{}
	err := r.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Ingress", "Ingress.Namespace", desired.Namespace, "Ingress.Name", desired.Name)
		if err := r.Create(ctx, desired); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		log.Info("Updating Ingress", "Ingress.Namespace", found.Namespace, "Ingress.Name", found.Name)
		desired.ResourceVersion = found.ResourceVersion
		if err := r.Update(ctx, desired); err != nil {
			return err
		}
	}
	return nil
}

// reconcileHPA 调和 HPA
func (r *InferenceServiceReconciler) reconcileHPA(ctx context.Context, isvc *aiv1.InferenceService, desired *autoscalingv2.HorizontalPodAutoscaler) error {
	log := log.FromContext(ctx)
	if desired == nil {
		return nil
	}

	found := &autoscalingv2.HorizontalPodAutoscaler{}
	err := r.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating HPA", "HPA.Namespace", desired.Namespace, "HPA.Name", desired.Name)
		if err := r.Create(ctx, desired); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		log.Info("Updating HPA", "HPA.Namespace", found.Namespace, "HPA.Name", found.Name)
		desired.ResourceVersion = found.ResourceVersion
		if err := r.Update(ctx, desired); err != nil {
			return err
		}
	}
	return nil
}

// updateStatus 更新 CR 状态
func (r *InferenceServiceReconciler) updateStatus(ctx context.Context, isvc *aiv1.InferenceService) error {
	log := log.FromContext(ctx)
	// 获取 Stable Deployment 状态
	deploy := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: isvc.Name + "-stable", Namespace: isvc.Namespace}, deploy); err != nil {
		return err
	}

	// 构造 URL
	var url string
	if isvc.Spec.Service != nil && isvc.Spec.Service.Host != "" {
		url = fmt.Sprintf("http://%s", isvc.Spec.Service.Host)
	} else {
		// 如果没有 Ingress，使用 Service 名称 (仅集群内访问)
		url = fmt.Sprintf("http://%s-stable.%s.svc.cluster.local", isvc.Name, isvc.Namespace)
	}

	// 更新 Status 字段
	isvc.Status.URL = url
	isvc.Status.ReadyReplicas = deploy.Status.ReadyReplicas
	isvc.Status.Ready = deploy.Status.ReadyReplicas > 0

	// 设置条件
	isvc.Status.Conditions = []metav1.Condition{
		{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             "DeploymentReady",
			Message:            "Inference service deployed successfully",
		},
	}

	log.Info("Updating InferenceService Status", "URL", url, "ReadyReplicas", deploy.Status.ReadyReplicas)
	return r.Status().Update(ctx, isvc)
}

// SetupWithManager sets up the controller with the Manager.
func (r *InferenceServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&aiv1.InferenceService{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.Ingress{}).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}).
		Complete(r)
}

// cleanupCanaryResources 删除灰度版本的 Deployment
func (r *InferenceServiceReconciler) cleanupCanaryResources(ctx context.Context, isvc *aiv1.InferenceService) error {

	name := fmt.Sprintf("%s-canary", isvc.Name)
	// 删除Deployment
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: isvc.Namespace,
		},
	}
	if err := r.Delete(ctx, dep); err != nil && !errors.IsNotFound(err) {
		return err
	}
	// 删除Service
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: isvc.Namespace,
		},
	}
	if err := r.Delete(ctx, service); err != nil && !errors.IsNotFound(err) {
		return err
	}
	// 删除Ingress
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: isvc.Namespace,
		},
	}
	if err := r.Delete(ctx, ingress); err != nil && !errors.IsNotFound(err) {
		return err
	}
	// 删除HPA
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: isvc.Namespace,
		},
	}
	if err := r.Delete(ctx, hpa); err != nil && !errors.IsNotFound(err) {
		return err
	}
	return client.IgnoreNotFound(nil)
}

// cleanupStableResources 删除稳定版本的 Deployment
func (r *InferenceServiceReconciler) cleanupStableResources(ctx context.Context, isvc *aiv1.InferenceService) error {
	name := fmt.Sprintf("%s-stable", isvc.Name)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: isvc.Namespace,
		},
	}
	if err := r.Delete(ctx, dep); err != nil && !errors.IsNotFound(err) {
		return err
	}
	// 删除Service
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: isvc.Namespace,
		},
	}
	if err := r.Delete(ctx, service); err != nil && !errors.IsNotFound(err) {
		return err
	}

	// 删除Ingress
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: isvc.Namespace,
		},
	}
	if err := r.Delete(ctx, ingress); err != nil && !errors.IsNotFound(err) {
		return err
	}

	// 删除HPA
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: isvc.Namespace,
		},
	}
	if err := r.Delete(ctx, hpa); err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

func (r *InferenceServiceReconciler) updateMetrics(isvc *aiv1.InferenceService) {
	metrics.InferenceReplicas.WithLabelValues(
		isvc.Spec.ModelName,
		isvc.Spec.ModelVersion,
		isvc.Name,
	).Set(float64(isvc.Status.ReadyReplicas))
	if isvc.Spec.Canary != nil && isvc.Spec.Canary.Enabled {
		metrics.InferenceReplicas.WithLabelValues(
			isvc.Spec.ModelName,
			isvc.Spec.ModelVersion,
			fmt.Sprintf("%s-canary", isvc.Name))
	}
}

func (r *InferenceServiceReconciler) setStatus(isvc *aiv1.InferenceService, s, reason, msg string) {
	isvc.Status.StableState = s
	isvc.Status.Conditions = []metav1.Condition{{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            msg,
	}}
}
