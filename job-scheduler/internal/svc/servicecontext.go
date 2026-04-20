// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package svc

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"k8s.io/client-go/kubernetes"
	k8sRest "k8s.io/client-go/rest"
	"kubeai-job-scheduler/internal/client"

	"kubeai-job-scheduler/internal/config"
	"kubeai-job-scheduler/internal/middleware"
	"kubeai-job-scheduler/internal/queue"
	"kubeai-job-scheduler/internal/scheduler"
)

type ServiceContext struct {
	Config                 config.Config
	RedisClient            *redis.Client
	ModelManagerClient     *client.ModelManagerClient
	InferenceGatewayClient *client.InferenceGatewayClient
	K8sClient              *kubernetes.Clientset
	K8sConfig              *k8sRest.Config
	InferenceQueue         *queue.TaskQueue
	TrainingQueue          *queue.TaskQueue
	ResourceTracker        *scheduler.ResourceTracker
	PlacementStrategy      scheduler.PlacementStrategy
	DeadLetterQueue        *queue.DeadLetterQueue
	MetricsMiddleware      rest.Middleware
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

	// K8s
	k8sConfig, err := k8sRest.InClusterConfig()
	if err != nil {
		logx.Must(err)
	}

	k8sClient, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		logx.Must(err)
	}

	// ModelManager Client
	modelClient := client.NewModelManagerClient(c.ModelManager.URL, c.ModelManager.Timeout)

	// 推理网关客户端
	inferenceClient := client.NewInferenceGatewayClient(c.InferenceGateway.URL, c.InferenceGateway.Timeout)

	// Queues
	inferenceQueue := queue.NewTaskQueue(rdb, c.Redis.Streams.Inference, c.Redis.ConsumerGroup)
	trainingQueue := queue.NewTaskQueue(rdb, c.Redis.Streams.Training, c.Redis.ConsumerGroup)
	deadLetterQueue := queue.NewDeadLetterQueue(rdb, c.Redis.Streams.Inference, c.Redis.ConsumerGroup)

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
		Config:                 c,
		RedisClient:            rdb,
		ModelManagerClient:     modelClient,
		InferenceGatewayClient: inferenceClient,
		K8sClient:              k8sClient,
		K8sConfig:              k8sConfig,
		InferenceQueue:         inferenceQueue,
		TrainingQueue:          trainingQueue,
		ResourceTracker:        resourceTracker,
		PlacementStrategy:      strategy,
		DeadLetterQueue:        deadLetterQueue,
		MetricsMiddleware:      middleware.NewMetricsMiddleware().Handle,
	}
}
