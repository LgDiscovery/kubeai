package svc

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	fiv1 "kubeai-inference-gateway/inferenceservice/api/v1"
	modelClient "kubeai-inference-gateway/internal/client"
	"kubeai-inference-gateway/internal/config"
	"kubeai-inference-gateway/internal/help"
	"kubeai-inference-gateway/internal/middleware"
	"kubeai-inference-gateway/internal/queue"
	"kubeai-inference-gateway/internal/repo"
	"kubeai-inference-gateway/internal/scheduler"
	tiv1 "kubeai-inference-gateway/trainingjob/api/v1"
	"log"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"time"
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
	DB                *gorm.DB                        // 数据库连接
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
	InferenceTaskRepo *repo.InferenceTaskRepo         // 推理任务 Repository
	TrainingTaskRepo  *repo.TrainingTaskRepo          // 训练任务 Repository
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化数据库
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host, c.Database.Port, c.Database.User, c.Database.Password,
		c.Database.DBName, c.Database.SSLMode)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "kubeai_",
			SingularTable: true,
		},
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Info,
				IgnoreRecordNotFoundError: true,
				ParameterizedQueries:      true,
				Colorful:                  false,
			},
		),
	})
	if err != nil {
		logx.Must(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxIdleConns(c.Database.MaxIdleConns)
	sqlDB.SetMaxOpenConns(c.Database.MaxOpenConns)

	// 初始化 Repository
	inferenceRepo := repo.NewInferenceTaskRepo(db)
	trainingRepo := repo.NewTrainingTaskRepo(db)

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
	k8sConfig, err := help.LoadKubeConfig()
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
	case "gpu-affinity":
		// 创建回退策略（可根据配置选择，这里默认使用 binpack）
		fallback := scheduler.NewBinpackStrategy(c.Scheduler.EnableGPUPacking, c.Scheduler.GPUBinpackWeight)
		// 使用 GPU 亲和策略，缓存标签前缀可从配置读取，未配置则使用默认值
		prefix := c.Scheduler.CacheLabelPrefix
		if prefix == "" {
			prefix = "kubeai.io/model-cache." // 默认前缀
		}
		strategy = scheduler.NewGPUAffinityStrategy(prefix, fallback)
	default:
		// 默认 binpack 策略
		strategy = scheduler.NewBinpackStrategy(c.Scheduler.EnableGPUPacking, c.Scheduler.GPUBinpackWeight)
	}

	return &ServiceContext{
		Config:            c,
		RedisClient:       rdb,
		DB:                db,
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
		InferenceTaskRepo: inferenceRepo,
		TrainingTaskRepo:  trainingRepo,
	}
}
