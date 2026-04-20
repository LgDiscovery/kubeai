package svc

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	k8srest "k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	aiv1 "kubeai-inference-gateway/api/v1"
	modelClient "kubeai-inference-gateway/internal/client"
	"kubeai-inference-gateway/internal/config"
	"kubeai-inference-gateway/internal/middleware"
	"kubeai-inference-gateway/internal/queue"
)

type ServiceContext struct {
	Config            config.Config
	RedisClient       *redis.Client
	K8sClient         *kubernetes.Clientset
	DynamicClient     dynamic.Interface
	CtrlClient        client.Client // controller-runtime client
	ModelMgrClient    *client.ModelManagerClient
	InferenceQueue    *queue.TaskQueue
	TrainingQueue     *queue.TaskQueue
	MetricsMiddleware rest.Middleware
	Mgr               manager.Manager // controller-runtime manager
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
	k8sConfig, err := k8srest.InClusterConfig()
	if err != nil {
		logx.Must(err)
	}
	k8sClient, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		logx.Must(err)
	}
	dynamicClient, err := dynamic.NewForConfig(k8sConfig)
	if err != nil {
		logx.Must(err)
	}

	// controller-runtime manager (用于 Operator)
	mgr, err := manager.New(k8sConfig, manager.Options{
		Scheme:             aiv1.Scheme,
		MetricsBindAddress: "0", // 禁用自带 metrics，使用 go-zero 暴露
	})
	if err != nil {
		logx.Must(err)
	}

	// ModelManager client
	modelClient := modelClient.NewModelManagerClient(c.ModelManager.URL, c.ModelManager.Timeout)

	// Queues
	inferenceQueue := queue.NewTaskQueue(rdb, c.Redis.Streams.Inference, c.Redis.ConsumerGroup)
	trainingQueue := queue.NewTaskQueue(rdb, c.Redis.Streams.Training, c.Redis.ConsumerGroup)

	metricsMiddleware := middleware.NewMetricsMiddleware().Handle

	return &ServiceContext{
		Config:            c,
		RedisClient:       rdb,
		K8sClient:         k8sClient,
		DynamicClient:     dynamicClient,
		CtrlClient:        mgr.GetClient(),
		ModelMgrClient:    modelClient,
		InferenceQueue:    inferenceQueue,
		TrainingQueue:     trainingQueue,
		MetricsMiddleware: metricsMiddleware,
		Mgr:               mgr,
	}
}
