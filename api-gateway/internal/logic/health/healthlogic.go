package health

import (
	"context"
	"time"

	"kubeai-api-gateway/internal/svc"
	"kubeai-api-gateway/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type HealthLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewHealthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HealthLogic {
	return &HealthLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Health 健康检查（liveness）
func (l *HealthLogic) Health() (resp *types.GeneralResponse, err error) {
	resp = &types.GeneralResponse{
		Code:    0,
		Message: "success",
		Data: map[string]interface{}{
			"status":    "UP",
			"service":   "api-gateway",
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}
	return
}

// CheckDependencies 检查所有依赖是否健康
func (l *HealthLogic) CheckDependencies() map[string]interface{} {
	result := make(map[string]interface{})

	// 检查数据库连接
	dbStatus := "UP"
	dbMsg := ""
	ctx, cancel := context.WithTimeout(l.ctx, 5*time.Second)
	defer cancel()

	sqlDB, err := l.svcCtx.DB.DB()
	if err != nil {
		dbStatus = "DOWN"
		dbMsg = err.Error()
	} else {
		if err := sqlDB.PingContext(ctx); err != nil {
			dbStatus = "DOWN"
			dbMsg = err.Error()
		}
	}
	result["database"] = map[string]interface{}{
		"status":  dbStatus,
		"message": dbMsg,
	}

	// 检查 Redis 连接
	redisStatus := "UP"
	redisMsg := ""
	if l.svcCtx.RedisClient != nil {
		redisCtx, redisCancel := context.WithTimeout(l.ctx, 5*time.Second)
		defer redisCancel()
		if err := l.svcCtx.RedisClient.Ping(redisCtx); err != nil {
			redisStatus = "DOWN"
			redisMsg = err.Error()
		}
	} else {
		redisStatus = "DOWN"
		redisMsg = "redis client not initialized"
	}
	result["redis"] = map[string]interface{}{
		"status":  redisStatus,
		"message": redisMsg,
	}

	// 检查下游服务配置
	result["upstreams"] = map[string]interface{}{
		"model_manager":     l.svcCtx.Config.Upstreams.ModelManager.URL,
		"job_scheduler":     l.svcCtx.Config.Upstreams.JobScheduler.URL,
		"inference_gateway": l.svcCtx.Config.Upstreams.InferenceGateway.URL,
	}

	return result
}
