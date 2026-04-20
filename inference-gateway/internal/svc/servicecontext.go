package svc

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	k8srest "k8s.io/client-go/rest"
	fiv1 "kubeai-inference-gateway/inferenceservice/api/v1"
	modelClient "kubeai-inference-gateway/internal/client"
	"kubeai-inference-gateway/internal/config"
	"kubeai-inference-gateway/internal/middleware"
	"kubeai-inference-gateway/internal/queue"
	"kubeai-inference-gateway/internal/scheduler"
	tiv1 "kubeai-inference-gateway/trainingjob/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(fiv1.AddToScheme(scheme))
	utilruntime.Must(tiv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

type ServiceContext struct {
	Config            config.Config                   // 配置文件
	RedisClient       *redis.Client                   // Redis 客户端
	K8sClient         *kubernetes.Clientset           // K8s 原生客户端
	DynamicClient     dynamic.Interface               // K8s 动态客户端
	CtrlClient        client.Client                   // controller-runtime 客户端
	ModelMgrClient    *modelClient.ModelManagerClient // 模型管理服务客户端
	InferenceQueue    *queue.TaskQueue                // 推理任务队列
	TrainingQueue     *queue.TaskQueue                // 训练任务队列
	MetricsMiddleware rest.Middleware                 // 监控指标中间件
	Mgr               manager.Manager                 // K8s Operator Manager
	ResourceTracker   *scheduler.ResourceTracker      // 资源跟踪器
	PlacementStrategy scheduler.PlacementStrategy     // 节点选择策略
	DeadLetterQueue   *queue.TaskQueue                // 死信队列
}

func NewServiceContext(c config.Config) *ServiceContext {
	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     c.Redis.Addr,
		Password: c.Redis.Password,
		DB:       c.Redis.DB,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		logx.Must(err)
	}

	// K8s config
	k8sConfig, err := k8srest.InClusterConfig() // 从 Pod 内部获取 K8s 配置
	if err != nil {
		logx.Must(err)
	}
	k8sClient, err := kubernetes.NewForConfig(k8sConfig) // 原生客户端
	if err != nil {
		logx.Must(err)
	}
	dynamicClient, err := dynamic.NewForConfig(k8sConfig) // 动态客户端
	if err != nil {
		logx.Must(err)
	}

	// controller-runtime manager (用于 Operator)
	mgr, err := manager.New(k8sConfig, manager.Options{
		Scheme: scheme,
		// 关闭 controller-runtime 自带 metrics
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
	})
	if err != nil {
		logx.Must(err)
	}

	// ModelManager client
	modelClient := modelClient.NewModelManagerClient(c.ModelManager.URL, c.ModelManager.Timeout)

	// Queues
	inferenceQueue := queue.NewTaskQueue(rdb, c.Redis.Streams.Inference, c.Redis.ConsumerGroup)
	trainingQueue := queue.NewTaskQueue(rdb, c.Redis.Streams.Training, c.Redis.ConsumerGroup)
	deadLetterQueue := queue.NewTaskQueue(rdb, c.Redis.Streams.DeadLetter, c.Redis.ConsumerGroup)

	metricsMiddleware := middleware.NewMetricsMiddleware().Handle

	// Resource Tracker
	resourceTracker := scheduler.NewResourceTracker(k8sClient, c.K8s.Namespace, c.ResourceSync.Interval)

	// Placement Strategy
	var strategy scheduler.PlacementStrategy
	switch c.Scheduler.Algorithm {
	case "spread":
		strategy = scheduler.NewSpreadStrategy()
	default:
		strategy = scheduler.NewBinpackStrategy(c.Scheduler.EnableGPUPacking, c.Scheduler.GPUBinpackWeight)
	}

	return &ServiceContext{
		Config:            c,
		RedisClient:       rdb,
		K8sClient:         k8sClient,
		DynamicClient:     dynamicClient,
		CtrlClient:        mgr.GetClient(),
		ModelMgrClient:    modelClient,
		InferenceQueue:    inferenceQueue,
		TrainingQueue:     trainingQueue,
		DeadLetterQueue:   deadLetterQueue,
		MetricsMiddleware: metricsMiddleware,
		Mgr:               mgr,
		ResourceTracker:   resourceTracker,
		PlacementStrategy: strategy,
	}
}
