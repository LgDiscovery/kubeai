// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

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
	"k8s.io/client-go/kubernetes"
	k8sRest "k8s.io/client-go/rest"
	"kubeai-job-scheduler/internal/client"
	"kubeai-job-scheduler/internal/model"
	"kubeai-job-scheduler/internal/repo"
	"log"
	"os"
	"time"

	"kubeai-job-scheduler/internal/config"
	"kubeai-job-scheduler/internal/middleware"
	"kubeai-job-scheduler/internal/queue"
)

type ServiceContext struct {
	Config             config.Config
	RedisClient        *redis.Client
	ModelManagerClient *client.ModelManagerClient
	DB                 *gorm.DB
	K8sClient          *kubernetes.Clientset
	K8sConfig          *k8sRest.Config
	InferenceQueue     *queue.TaskQueue
	TrainingQueue      *queue.TaskQueue
	MetricsMiddleware  rest.Middleware
	InferenceTaskRepo  *repo.InferenceTaskRepo
	TrainingTaskRepo   *repo.TrainingTaskRepo
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

	// 自动迁移
	if err := db.AutoMigrate(&model.InferenceTask{}, &model.TrainingTask{}); err != nil {
		logx.Must(err)
	}

	// 初始化 Repository
	inferenceTaskRepo := repo.NewInferenceTaskRepo(db)
	trainingTaskRepo := repo.NewTrainingTaskRepo(db)

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

	// Queues
	inferenceQueue := queue.NewTaskQueue(rdb, c.Redis.Streams.Inference, c.Redis.ConsumerGroup)
	trainingQueue := queue.NewTaskQueue(rdb, c.Redis.Streams.Training, c.Redis.ConsumerGroup)

	return &ServiceContext{
		Config:             c,
		RedisClient:        rdb,
		ModelManagerClient: modelClient,
		DB:                 db,
		K8sClient:          k8sClient,
		K8sConfig:          k8sConfig,
		InferenceQueue:     inferenceQueue,
		TrainingQueue:      trainingQueue,
		InferenceTaskRepo:  inferenceTaskRepo,
		TrainingTaskRepo:   trainingTaskRepo,
		MetricsMiddleware:  middleware.NewMetricsMiddleware().Handle,
	}
}
