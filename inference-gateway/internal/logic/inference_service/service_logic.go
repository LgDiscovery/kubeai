package inference_service

import (
	"context"
	"fmt"
	"time"

	fiv1 "kubeai-inference-gateway/inferenceservice/api/v1"
	"kubeai-inference-gateway/internal/svc"
	"kubeai-inference-gateway/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ListInferenceServiceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListInferenceServiceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListInferenceServiceLogic {
	return &ListInferenceServiceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ListInferenceService 列出所有推理服务
func (l *ListInferenceServiceLogic) ListInferenceService() (resp *types.InferenceServiceListResp, err error) {
	var isvcList fiv1.InferenceServiceList
	if err := l.svcCtx.CtrlClient.List(l.ctx, &isvcList, client.InNamespace(l.svcCtx.Config.K8s.Namespace)); err != nil {
		l.Logger.Errorf("List InferenceService failed: %v", err)
		return nil, fmt.Errorf("获取推理服务列表失败: %w", err)
	}

	items := make([]types.InferenceServiceItem, 0, len(isvcList.Items))
	for _, isvc := range isvcList.Items {
		items = append(items, convertToItem(&isvc))
	}

	resp = &types.InferenceServiceListResp{
		Code:    0,
		Message: "success",
		Data: types.InferenceServiceListData{
			Total: len(items),
			Items: items,
		},
	}
	return
}

// convertToItem 将 CRD 对象转换为响应类型
func convertToItem(isvc *fiv1.InferenceService) types.InferenceServiceItem {
	replicas := int32(1)
	if isvc.Spec.Replicas != nil {
		replicas = *isvc.Spec.Replicas
	}

	item := types.InferenceServiceItem{
		Name:          isvc.Name,
		ModelName:     isvc.Spec.ModelName,
		ModelVersion:  isvc.Spec.ModelVersion,
		Image:         isvc.Spec.Image,
		Replicas:      replicas,
		ReadyReplicas: isvc.Status.ReadyReplicas,
		Status:        isvc.Status.StableState,
		URL:           isvc.Status.URL,
		CreatedAt:     isvc.CreationTimestamp.Format(time.RFC3339),
	}

	// 资源配置
	if cpu, ok := isvc.Spec.Resources.Requests["cpu"]; ok {
		item.CPU = cpu.String()
	}
	if mem, ok := isvc.Spec.Resources.Requests["memory"]; ok {
		item.Memory = mem.String()
	}
	if gpu, ok := isvc.Spec.Resources.Requests["nvidia.com/gpu"]; ok {
		item.GPU = gpu.String()
	}

	// 灰度配置
	if isvc.Spec.Canary != nil && isvc.Spec.Canary.Enabled {
		item.CanaryEnabled = true
		item.CanaryTraffic = isvc.Spec.Canary.Weight
	}

	return item
}

// GetInferenceService 获取单个推理服务详情
type GetInferenceServiceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetInferenceServiceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetInferenceServiceLogic {
	return &GetInferenceServiceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetInferenceServiceLogic) GetInferenceService(name string) (resp *types.InferenceServiceDetailResp, err error) {
	var isvc fiv1.InferenceService
	if err := l.svcCtx.CtrlClient.Get(l.ctx, client.ObjectKey{
		Namespace: l.svcCtx.Config.K8s.Namespace,
		Name:      name,
	}, &isvc); err != nil {
		l.Logger.Errorf("Get InferenceService %s failed: %v", name, err)
		return nil, fmt.Errorf("获取推理服务详情失败: %w", err)
	}

	item := convertToItem(&isvc)
	resp = &types.InferenceServiceDetailResp{
		Code:    0,
		Message: "success",
		Data:    item,
	}
	return
}

// DeleteInferenceService 删除推理服务
type DeleteInferenceServiceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteInferenceServiceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteInferenceServiceLogic {
	return &DeleteInferenceServiceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteInferenceServiceLogic) DeleteInferenceService(name string) (resp *types.CommonResp, err error) {
	var isvc fiv1.InferenceService
	isvc.Name = name
	isvc.Namespace = l.svcCtx.Config.K8s.Namespace

	if err := l.svcCtx.CtrlClient.Delete(l.ctx, &isvc); err != nil {
		l.Logger.Errorf("Delete InferenceService %s failed: %v", name, err)
		return nil, fmt.Errorf("删除推理服务失败: %w", err)
	}

	resp = &types.CommonResp{
		Code:    0,
		Message: "推理服务删除成功",
	}
	return
}

// CreateInferenceService 创建推理服务
type CreateInferenceServiceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateInferenceServiceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateInferenceServiceLogic {
	return &CreateInferenceServiceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateInferenceServiceLogic) CreateInferenceService(req *types.CreateInferenceServiceReq) (resp *types.InferenceServiceDetailResp, err error) {
	// 构建资源需求
	resources, err := buildResourceRequirements(req.CPU, req.Memory, req.GPU)
	if err != nil {
		return nil, fmt.Errorf("资源配置无效: %w", err)
	}

	replicas := int32(req.Replicas)
	port := int32(8501)
	if req.Port > 0 {
		port = int32(req.Port)
	}

	isvc := &fiv1.InferenceService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: l.svcCtx.Config.K8s.Namespace,
		},
		Spec: fiv1.InferenceServiceSpec{
			ModelName:    req.ModelName,
			ModelVersion: req.ModelVersion,
			Image:        req.Image,
			Port:         port,
			Replicas:     &replicas,
			Resources:    resources,
			Service: &fiv1.ServiceSpec{
				Type:       "ClusterIP",
				Port:       port,
				TargetPort: port,
			},
		},
	}

	// 灰度发布配置
	if req.CanaryEnabled {
		isvc.Spec.Canary = &fiv1.CanarySpec{
			Enabled:      true,
			Version:      req.ModelVersion + "-canary",
			Weight:       int32(req.CanaryTraffic),
			ModelName:    req.ModelName,
			ModelVersion: req.ModelVersion,
		}
	}

	// 自动扩缩容配置
	if req.EnableAutoscaling {
		maxReplicas := int32(req.MaxReplicas)
		targetCPU := int32(80)
		targetMem := int32(80)
		isvc.Spec.Autoscaling = &fiv1.AutoscalingSpec{
			MinReplicas:              &replicas,
			MaxReplicas:              maxReplicas,
			TargetCPUUtilization:     &targetCPU,
			TargetMemoryUtilization:  &targetMem,
		}
	}

	if err := l.svcCtx.CtrlClient.Create(l.ctx, isvc); err != nil {
		l.Logger.Errorf("Create InferenceService failed: %v", err)
		return nil, fmt.Errorf("创建推理服务失败: %w", err)
	}

	item := convertToItem(isvc)
	resp = &types.InferenceServiceDetailResp{
		Code:    0,
		Message: "推理服务创建成功",
		Data:    item,
	}
	return
}
