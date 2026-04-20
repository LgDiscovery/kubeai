package svc

import (
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"kubeai-model-manager/internal/config"
	"kubeai-model-manager/internal/middleware"
	"kubeai-model-manager/internal/model"
	"kubeai-model-manager/internal/repo"
	"kubeai-model-manager/internal/storage"
	"log"
	"os"
	"time"
)

type ServiceContext struct {
	Config            config.Config
	DB                *gorm.DB
	MinIOClient       *storage.MinIOClient
	ModelRepo         *repo.ModelRepo
	VersionRepo       *repo.VersionRepo
	MetricsMiddleware rest.Middleware
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
	if err := db.AutoMigrate(&model.Model{}, &model.ModelVersion{}); err != nil {
		logx.Must(err)
	}

	// 初始化 MinIO 客户端
	storageClient, err := storage.NewMinIOClient(c.MinIO)
	if err != nil {
		logx.Must(err)
	}

	return &ServiceContext{
		Config:            c,
		DB:                db,
		MinIOClient:       storageClient,
		ModelRepo:         repo.NewModelRepo(db),
		VersionRepo:       repo.NewVersionRepo(db),
		MetricsMiddleware: middleware.NewMetricsMiddleware().Handle,
	}
}
