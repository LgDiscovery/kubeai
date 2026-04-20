// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package svc

import (
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"kubeai-api-gateway/internal/config"
	"kubeai-api-gateway/internal/discovery"
	"kubeai-api-gateway/internal/middleware"
	"kubeai-api-gateway/internal/model"
	"kubeai-api-gateway/pkg/jwt"
	"log"
	"os"
	"sync"
	"time"
)

type ServiceContext struct {
	Config              config.Config
	DB                  *gorm.DB
	RedisClient         *redis.Redis
	MetricsMiddleware   rest.Middleware
	AuthMiddleware      rest.Middleware
	RateLimitMiddleware rest.Middleware
	CorsMiddleware      rest.Middleware

	// 热配置相关
	hotConfigMu sync.RWMutex
	HotConfig   config.HotConfig
	Discovery   *discovery.ServiceDiscovery
}

func NewServiceContext(c config.Config) *ServiceContext {

	// 0.初始化服务发现
	sd := discovery.NewServiceDiscovery(
		c.Etcd.Hosts,
		c.Discovery.ModelManagerKey,
		c.Discovery.TaskSchedulerKey,
		c.Discovery.InferenceGatewayKey,
		c.Discovery.ObservabilityGatewayKey,
	)

	sd.Watch()

	// 1. 初始化数据库
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
	if err := db.AutoMigrate(&model.User{}); err != nil {
		logx.Must(err)
	}

	// 2. 初始化Redis客户端 分布式限流
	redisClient, err := redis.NewRedis(c.RedisConf)
	if err != nil {
		logx.Errorf("init redis client for rate limit failed: %v", err)
	}

	// 3. 初始化 JWT 配置
	jwt.SetConfig(c.Auth.AccessSecret, int64(c.Auth.AccessExpire))

	svcCtx := &ServiceContext{
		Config:              c,
		DB:                  db,
		RedisClient:         redisClient,
		MetricsMiddleware:   middleware.NewMetricsMiddleware().Handle,
		AuthMiddleware:      middleware.NewAuthMiddleware().Handle,
		RateLimitMiddleware: middleware.NewRateLimitMiddleware(c.RateLimit, redisClient).Handle,
		CorsMiddleware:      middleware.NewCorsMiddleware().Handle,
		Discovery:           sd,
	}

	// 启动配置热更新
	if len(c.Etcd.Hosts) > 0 && c.Etcd.Key != "" {
		if err := config.StartConfigListener(c.Etcd, c.Etcd.Key, svcCtx.onHotConfigChanged); err != nil {
			logx.Errorf("start config listener failed: %v", err)
		}

	}
	return svcCtx
}

// onHotConfigChanged 配置变更回调
func (svcCtx *ServiceContext) onHotConfigChanged(hotCfg config.HotConfig) {
	svcCtx.hotConfigMu.Lock()
	defer svcCtx.hotConfigMu.Unlock()
	svcCtx.HotConfig = hotCfg

	// 更新 JWT 配置
	jwt.SetConfig(hotCfg.Auth.AccessSecret, hotCfg.Auth.AccessExpire)

	// 更新日志级别
	if err := logx.SetUp(logx.LogConf{Level: hotCfg.Log.Level}); err != nil {
		logx.Errorf("update log level failed: %v", err)
	}
}
